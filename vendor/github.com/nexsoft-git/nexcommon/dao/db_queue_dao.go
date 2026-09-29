package dao

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/nexsoft-git/nexcommon/model"
	"github.com/nexsoft-git/nexlogger/log"
)

const acknowledged_status = "acknowledged"
const waitingack_status = "waiting_ack"
const done_processed_status = "done_processed"
const error_processed_status = "error_processed"

type DBQueueDAO interface {
	SetPendingErrorDuration(
		time time.Duration,
	)

	EnablePendingProcessingMessage()

	EnableDynamicNumberOfProcessor()

	InsertToDBQueueDAO(
		payload model.DBQueueModel,
	) (
		err error,
	)

	UpdateWaitingForAckDBQueue(
		tx *sql.Tx,
		dto model.DBQueueModel,
	) (
		err error,
	)

	UpdateWaitingForDataCompletion(
		tx *sql.Tx,
		dto model.DBQueueModel,
	) (
		err error,
	)

	UpdateErrorProcessedDBQueue(
		tx *sql.Tx,
		dto model.DBQueueModel,
	) (
		err error,
	)

	DeleteAcknowledgedMessage(
		messageID string,
	) (
		err error,
	)
	DeleteFinishProcessedDBQueue(
		tx *sql.Tx,
		dto model.DBQueueModel,
	) (
		err error,
	)

	DeleteExpiredQueueData(
		expired int,
	) (
		err error,
	)

	GetFromDBQueueDAO(
		queueType string,
		length int,
	) (
		tx *sql.Tx,
		result []model.DBQueueModel,
		err error,
	)

	GetDistinctUniqueIdentifierFromDBQueueDAO(
		queueType string,
	) (
		result []string,
		err error,
	)

	GetFromDBQueueDAOByUniqueIdentifier(
		queueType string,
		uniqueIdentifier string,
		length int,
	) (
		tx *sql.Tx,
		result []model.DBQueueModel,
		err error,
	)
}

func NewDBQueueDAO(
	db *sql.DB,
	schema string,
	table string,
	maxRetry int,
	ackWaiting time.Duration,
	historyEnabled bool,
) DBQueueDAO {
	return &dbQueueDAO{
		db:                     db,
		schema:                 schema,
		table:                  table,
		maxRetry:               maxRetry,
		ackWaiting:             ackWaiting,
		historyEnabled:         historyEnabled,
		durationPendingIfError: 0 * time.Second,
	}
}

type dbQueueDAO struct {
	db                             *sql.DB
	schema                         string
	table                          string
	maxRetry                       int
	ackWaiting                     time.Duration
	historyEnabled                 bool
	durationPendingIfError         time.Duration
	enablePendingProcessingMessage bool
	enableDynamicNumberOfProcessor bool
}

func (d *dbQueueDAO) SetPendingErrorDuration(
	time time.Duration,
) {
	d.durationPendingIfError = time
}

func (d *dbQueueDAO) EnablePendingProcessingMessage() {
	d.enablePendingProcessingMessage = true
}

func (d *dbQueueDAO) EnableDynamicNumberOfProcessor() {
	d.enableDynamicNumberOfProcessor = true
}

func (d *dbQueueDAO) getDBTable() string {
	if d.schema == "" {
		return d.table
	} else {
		return fmt.Sprintf("%s.%s", d.schema, d.table)
	}
}

func (d *dbQueueDAO) getHistoryTable() string {
	if d.schema == "" {
		return "history_" + d.table
	} else {
		return fmt.Sprintf("%s.history_%s", d.schema, d.table)
	}
}

func (d dbQueueDAO) InsertToDBQueueDAO(
	payload model.DBQueueModel,
) (
	err error,
) {

	tx, err := d.db.Begin()

	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			_ = tx.Commit()
		}
	}()

	queryInsert := " type, payload, created_at, updated_at, retry_counter, max_retry, message_id, ack_topic, incoming_message_id "

	param := []interface{}{
		payload.QueueType.String,
		payload.Payload.String,
		payload.CreatedAt.Time,
		payload.UpdatedAt.Time,
		d.maxRetry,
		payload.MessageID.String,
		payload.AckTopic.String,
		payload.IncomingMessageID,
	}

	queryParamInsert := " $1, $2, $3, $4, 0, $5, $6, $7, $8 "

	if d.enablePendingProcessingMessage {
		queryInsert += ", processed_after "
		queryParamInsert += fmt.Sprintf(", $%d ", len(param)+1)
		param = append(param, payload.ProcessedAfter)
	}

	if d.enableDynamicNumberOfProcessor {
		queryInsert += ", unique_identifier "
		queryParamInsert += fmt.Sprintf(", $%d ", len(param)+1)
		param = append(param, payload.UniqueIdentifier)
	}

	stmt, err := tx.Prepare(
		fmt.Sprintf(
			"INSERT INTO %s "+
				"( %s ) "+
				" VALUES "+
				"( %s ) ON CONFLICT DO NOTHING ", d.getDBTable(), queryInsert, queryParamInsert),
	)

	if err != nil {
		return
	}

	_, err = stmt.Exec(param...)

	if err != nil {
		return
	}

	if d.historyEnabled {
		return d.doInsertHistoryQueue(tx, payload)
	}

	return nil
}

func (d dbQueueDAO) UpdateWaitingForAckDBQueue(
	tx *sql.Tx,
	dto model.DBQueueModel,
) (
	err error,
) {

	stmt, err := tx.Prepare(
		fmt.Sprintf(
			"UPDATE %s SET "+
				" ack_waiting = true, "+
				" updated_at = $2, "+
				" retry_counter = 0, "+
				" process_message = '' "+
				"WHERE "+
				" id = $1 ",
			d.getDBTable(),
		),
	)

	if err != nil {
		return
	}

	_, err = stmt.Exec(
		[]interface{}{
			dto.ID.Int64,
			dto.UpdatedAt.Time,
		}...,
	)

	if err != nil {
		return
	}

	if d.historyEnabled {
		return d.doUpdateHistoryQueueStatus(tx, dto, waitingack_status)
	}

	return nil
}

func (d dbQueueDAO) UpdateWaitingForDataCompletion(
	tx *sql.Tx,
	dto model.DBQueueModel,
) (
	err error,
) {

	addition := ""
	param := []interface{}{
		dto.UpdatedAt.Time,
		dto.ProcessMessage.String,
		dto.ID.Int64,
	}

	if d.enablePendingProcessingMessage {
		addition = ", processed_after = $4 "
		param = append(param, time.Now().Add(d.durationPendingIfError))
	}

	stmt, err := tx.Prepare(
		fmt.Sprintf(
			"UPDATE %s SET "+
				" updated_at = $1, "+
				" process_message = $2 %s "+
				"WHERE "+
				" id = $3 ",
			d.getDBTable(), addition,
		),
	)

	if err != nil {
		return
	}

	_, err = stmt.Exec(
		param...,
	)

	if err != nil {
		return
	}

	if d.historyEnabled {
		return d.doUpdateHistoryQueueStatus(tx, dto, done_processed_status)
	}

	return nil
}

func (d dbQueueDAO) UpdateErrorProcessedDBQueue(
	tx *sql.Tx,
	dto model.DBQueueModel,
) (
	err error,
) {

	stmt, err := tx.Prepare(
		fmt.Sprintf(
			"UPDATE %s SET "+
				" retry_counter = retry_counter + 1, "+
				" process_message = $1, "+
				" updated_at = $3 "+
				"WHERE "+
				" id = $2 ",
			d.getDBTable(),
		),
	)

	if err != nil {
		return
	}

	_, err = stmt.Exec([]interface{}{dto.ProcessMessage.String, dto.ID.Int64, dto.UpdatedAt.Time}...)
	if err != nil {
		return
	}

	if d.historyEnabled {
		return d.doUpdateHistoryQueueStatus(tx, dto, error_processed_status)
	}

	return nil
}

func (d dbQueueDAO) DeleteAcknowledgedMessage(
	messageID string,
) (
	err error,
) {

	tx, err := d.db.Begin()

	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			_ = tx.Commit()
		}
	}()

	var param []interface{}

	param = append(param, []interface{}{messageID}...)

	stmt, err := tx.Prepare(
		fmt.Sprintf(
			"DELETE FROM %s "+
				" WHERE message_id = $1 ",
			d.getDBTable(),
		),
	)

	if err != nil {
		return
	}

	_, err = stmt.Exec(param...)
	if err != nil {
		return
	}

	if d.historyEnabled {
		return d.doUpdateHistoryQueueStatus(tx, model.DBQueueModel{
			MessageID: sql.NullString{String: messageID},
			UpdatedAt: sql.NullTime{Time: time.Now()},
		}, acknowledged_status)
	}

	return nil
}

func (d dbQueueDAO) DeleteFinishProcessedDBQueue(
	tx *sql.Tx,
	dto model.DBQueueModel,
) (
	err error,
) {

	var param []interface{}

	param = append(param, []interface{}{dto.ID.Int64}...)

	stmt, err := tx.Prepare(
		fmt.Sprintf(
			"DELETE FROM %s "+
				" WHERE id = $1 ",
			d.getDBTable(),
		),
	)

	if err != nil {
		return
	}

	_, err = stmt.Exec(param...)
	if err != nil {
		return
	}

	if d.historyEnabled {
		return d.doUpdateHistoryQueueStatus(tx, model.DBQueueModel{
			MessageID:      sql.NullString{String: dto.MessageID.String},
			ProcessMessage: dto.ProcessMessage,
			UpdatedAt:      sql.NullTime{Time: time.Now()},
		}, done_processed_status)
	}

	return nil
}

func (d dbQueueDAO) DeleteExpiredQueueData(
	expired int,
) (
	err error,
) {

	query := fmt.Sprintf(
		" DELETE FROM %s "+
			" WHERE DATE_PART('day', NOW() - created_at) >= $1 ", d.getDBTable())

	stmt, err := d.db.Prepare(
		query,
	)

	if err != nil {
		return
	}

	_, err = stmt.Exec(expired)
	if err != nil {
		return
	}

	return nil
}

func (d dbQueueDAO) GetFromDBQueueDAO(
	queueType string,
	length int,
) (
	tx *sql.Tx,
	result []model.DBQueueModel,
	err error,
) {

	tx, err = d.db.Begin()
	if err != nil {
		return
	}

	timeNow := time.Now()

	queryWhere := " type = $1 AND ack_waiting IS FALSE AND retry_counter <= max_retry "

	if d.durationPendingIfError.Seconds() > 0 {
		queryWhere += " AND (EXTRACT(EPOCH from $4 - updated_at) > $5 OR retry_counter = 0 ) "
	}

	if d.enablePendingProcessingMessage {
		queryWhere += "AND (processed_after is NULL OR processed_after <= $4) "
	}

	query := fmt.Sprintf(
		"SELECT "+
			" id, type, payload, message_id, ack_topic "+
			" FROM %s "+
			" WHERE "+
			"	( %s )  "+
			"	OR ( type = $1 AND ack_waiting IS TRUE AND (EXTRACT(EPOCH from $4 - updated_at) > $3 )) "+
			" ORDER BY updated_at ASC "+
			" LIMIT $2 "+
			" FOR UPDATE SKIP LOCKED ", d.getDBTable(), queryWhere)

	if d.enableDynamicNumberOfProcessor {
		query = fmt.Sprintf(
			"SELECT "+
				" id, type, payload, message_id, ack_topic, unique_identifier "+
				" FROM %s "+
				" WHERE "+
				"	( %s )  "+
				"	OR ( type = $1 AND ack_waiting IS TRUE AND (EXTRACT(EPOCH from $4 - updated_at) > $3 )) "+
				" ORDER BY updated_at ASC "+
				" LIMIT $2 "+
				" FOR UPDATE SKIP LOCKED ", d.getDBTable(), queryWhere)
	}

	param := []interface{}{
		queueType,
		length,
		d.ackWaiting.Seconds(),
		timeNow,
	}

	if d.durationPendingIfError.Seconds() > 0 {
		param = append(param, d.durationPendingIfError.Seconds())
	}

	rows, err := tx.Query(
		query,
		param...,
	)

	if err != nil {
		return
	}

	if rows != nil {
		defer func() {
			errs := rows.Close()
			if errs != nil {
				log.Error().
					Err(errs).
					Caller().
					Msg("Error Found when close get list dao connection")
				return
			}
		}()

		for rows.Next() {

			var temp model.DBQueueModel

			if d.enableDynamicNumberOfProcessor {
				err = rows.Scan(&temp.ID, &temp.QueueType, &temp.Payload, &temp.MessageID, &temp.AckTopic, &temp.UniqueIdentifier)
			} else {
				err = rows.Scan(&temp.ID, &temp.QueueType, &temp.Payload, &temp.MessageID, &temp.AckTopic)
			}
			if err != nil {
				return
			}

			result = append(result, temp)

		}
	}

	return

}

func (d dbQueueDAO) GetDistinctUniqueIdentifierFromDBQueueDAO(
	queueType string,
) (
	result []string,
	err error,
) {
	rows, err := d.db.Query(
		fmt.Sprintf(
			"SELECT DISTINCT unique_identifier FROM %s WHERE type = $1 AND unique_identifier IS NOT NULL AND unique_identifier != ''",
			d.getDBTable(),
		),
		queueType,
	)
	if err != nil {
		return
	}

	if rows != nil {
		defer func() {
			errs := rows.Close()
			if errs != nil {
				log.Error().
					Err(errs).
					Caller().
					Msg("Error Found when close get unique identifier dao connection")
			}
		}()

		for rows.Next() {
			var uniqueIdentifier sql.NullString

			err = rows.Scan(&uniqueIdentifier)
			if err != nil {
				return
			}

			if uniqueIdentifier.Valid && uniqueIdentifier.String != "" {
				result = append(result, uniqueIdentifier.String)
			}
		}
	}

	return
}

func (d dbQueueDAO) GetFromDBQueueDAOByUniqueIdentifier(
	queueType string,
	uniqueIdentifier string,
	length int,
) (
	tx *sql.Tx,
	result []model.DBQueueModel,
	err error,
) {
	tx, err = d.db.Begin()
	if err != nil {
		return
	}

	timeNow := time.Now()

	queryWhere := " type = $1 AND unique_identifier = $2 AND ack_waiting IS FALSE AND retry_counter <= max_retry "

	if d.durationPendingIfError.Seconds() > 0 {
		queryWhere += " AND (EXTRACT(EPOCH from $5 - updated_at) > $6 OR retry_counter = 0 ) "
	}

	if d.enablePendingProcessingMessage {
		queryWhere += "AND (processed_after is NULL OR processed_after <= $5) "
	}

	query := fmt.Sprintf(
		"SELECT "+
			" id, type, payload, message_id, ack_topic, unique_identifier "+
			" FROM %s "+
			" WHERE "+
			"	( %s )  "+
			"	OR ( type = $1 AND unique_identifier = $2 AND ack_waiting IS TRUE AND (EXTRACT(EPOCH from $5 - updated_at) > $4 )) "+
			" ORDER BY updated_at ASC "+
			" LIMIT $3 "+
			" FOR UPDATE SKIP LOCKED ", d.getDBTable(), queryWhere)

	param := []interface{}{
		queueType,
		uniqueIdentifier,
		length,
		d.ackWaiting.Seconds(),
		timeNow,
	}

	if d.durationPendingIfError.Seconds() > 0 {
		param = append(param, d.durationPendingIfError.Seconds())
	}

	rows, err := tx.Query(
		query,
		param...,
	)
	if err != nil {
		return
	}

	if rows != nil {
		defer func() {
			errs := rows.Close()
			if errs != nil {
				log.Error().
					Err(errs).
					Caller().
					Msg("Error Found when close get list by unique identifier dao connection")
			}
		}()

		for rows.Next() {
			var temp model.DBQueueModel

			err = rows.Scan(&temp.ID, &temp.QueueType, &temp.Payload, &temp.MessageID, &temp.AckTopic, &temp.UniqueIdentifier)
			if err != nil {
				return
			}

			result = append(result, temp)
		}
	}

	return
}

func (d dbQueueDAO) doInsertHistoryQueue(
	tx *sql.Tx,
	payload model.DBQueueModel,
) (
	err error,
) {
	stmt, err := tx.Prepare(
		fmt.Sprintf(
			"INSERT INTO %s "+
				"( type, payload, created_at, updated_at, message_id, incoming_message_id, status ) "+
				" VALUES "+
				"( $1, $2, $3, $4, $5, $6, $7 ) ON CONFLICT DO NOTHING ", d.getHistoryTable(),
		),
	)

	if err != nil {
		return
	}

	_, err = stmt.Exec(
		[]interface{}{
			payload.QueueType.String,
			payload.Payload.String,
			payload.CreatedAt.Time,
			payload.UpdatedAt.Time,
			payload.MessageID.String,
			payload.IncomingMessageID,
			"new",
		}...,
	)

	return err
}

func (d dbQueueDAO) doUpdateHistoryQueueStatus(
	tx *sql.Tx,
	payload model.DBQueueModel,
	status string,
) (
	err error,
) {

	if status == error_processed_status {
		payload.ProcessMessage.Valid = true
	}

	stmt, err := tx.Prepare(
		fmt.Sprintf(
			"UPDATE %s "+
				" set status = $1, updated_at = $2, process_message = $4 "+
				" WHERE message_id = $3", d.getHistoryTable(),
		),
	)

	if err != nil {
		return
	}

	_, err = stmt.Exec(
		[]interface{}{
			status,
			payload.UpdatedAt.Time,
			payload.MessageID.String,
			payload.ProcessMessage,
		}...,
	)

	return err
}
