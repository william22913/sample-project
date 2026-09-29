package util

import (
	"context"
	"net/url"
	"os"
	"strings"

	"github.com/Azure/azure-pipeline-go/pipeline"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/nexsoft-git/nexcommon/dto/in"
)

func NewAzureConfig(
	accountKey string,
	accountName string,
	host string,
	suffix string,
) AzureConfig {
	return AzureConfig{
		accountKey:  accountKey,
		accountName: accountName,
		host:        host,
		suffix:      suffix,
	}
}

type AzureConfig struct {
	accountKey  string
	accountName string
	host        string
	suffix      string
}

func (a AzureConfig) GeneratePipeline() pipeline.Pipeline {
	credential, err := azblob.NewSharedKeyCredential(a.accountName, a.accountKey)
	if err != nil {
		//TODO LOG FATAL
		return nil
	}

	return azblob.NewPipeline(credential, azblob.PipelineOptions{})

}

func NewAzureHelper(
	config AzureConfig,
) AzureHelper {
	return AzureHelper{
		config:   config,
		pipeline: config.GeneratePipeline(),
	}
}

type AzureHelper struct {
	config   AzureConfig
	pipeline pipeline.Pipeline
}

func (a AzureHelper) UploadFileToAzure(
	files []in.MultipartFile,
	dirPrefix string,
) (
	azureUrl []string,
	err error,
) {
	ctx := context.Background()

	if dirPrefix != "" {
		dirPrefix = strings.ReplaceAll(dirPrefix, "\\", "/")
		dirPrefix = strings.Trim(dirPrefix, "/")
		dirPrefix += "/"
	}

	for i := 0; i < len(files); i++ {
		containerURL := a.GetContainer()
		temp := files[i]

		if temp.Header == nil {
			continue
		}

		var path string

		path = temp.Header.Filename
		if temp.Alias != "" {
			path = temp.Alias
		}

		if string(containerURL.String()[len(containerURL.String())-1]) == "/" {
			if string(path[0]) == "/" {
				path = path[1:]
			}
		} else {
			if string(path[0]) != "/" {
				path += "/" + path
			}
		}

		content, err := os.OpenFile(temp.FullPath, os.O_RDONLY, os.ModeAppend)
		if err != nil {
			return nil, err
		}

		path = dirPrefix + path
		blobURL := containerURL.NewBlockBlobURL(path)

		_, err = azblob.UploadFileToBlockBlob(
			ctx,
			content,
			blobURL,
			azblob.UploadToBlockBlobOptions{
				BlockSize:   4 * 1024 * 1024,
				Parallelism: 16,
			},
		)

		if err != nil {
			return nil, err
		}

		azureUrl = append(azureUrl, blobURL.String())

	}

	return azureUrl, nil
}

func (a AzureHelper) GetContainer() azblob.ContainerURL {
	URL, _ := url.Parse(a.config.host + a.config.suffix)

	containerURL := azblob.NewContainerURL(
		*URL,
		a.pipeline,
	)

	return containerURL
}

func (a AzureHelper) GetFileProperties(
	path string,
) (
	*azblob.BlobGetPropertiesResponse,
	error,
) {
	ctx := context.Background()
	URL, _ := url.Parse(path)

	blobURL := azblob.NewBlockBlobURL(
		*URL,
		a.pipeline,
	)

	return blobURL.GetProperties(ctx, azblob.BlobAccessConditions{}, azblob.ClientProvidedKeyOptions{})

}

func (a AzureHelper) DeleteFileFromCDN(
	files []string,
) {
	ctx := context.Background()
	for i := 0; i < len(files); i++ {
		URL, _ := url.Parse(a.config.host + a.config.suffix)
		containerURL := azblob.NewContainerURL(
			*URL,
			a.pipeline,
		)

		path := strings.ReplaceAll(files[i], a.config.host+a.config.suffix, "")
		blobURL := containerURL.NewBlockBlobURL(path)
		_, _ = blobURL.Delete(ctx, azblob.DeleteSnapshotsOptionNone, azblob.BlobAccessConditions{})
	}
}

type deletionError map[string]error

func (a AzureHelper) DeleteAllFilesInContainer(
	containerPath string,
) deletionError {
	errResult := make(deletionError)
	ctx := context.Background()
	URL, _ := url.Parse(a.config.host + a.config.suffix)

	containerURL := azblob.NewContainerURL(
		*URL,
		a.pipeline,
	)

	for marker := (azblob.Marker{}); marker.NotDone(); {
		listBlob, err := containerURL.ListBlobsFlatSegment(ctx, marker, azblob.ListBlobsSegmentOptions{
			Prefix: containerPath + "/",
		})

		if err != nil {
			errResult["blob"] = err
			return errResult
		}

		// Delete each blob in the segment
		for _, blob := range listBlob.Segment.BlobItems {
			blobURL := containerURL.NewBlockBlobURL(blob.Name)
			_, err := blobURL.Delete(ctx, azblob.DeleteSnapshotsOptionNone, azblob.BlobAccessConditions{})
			if err != nil {
				errResult[blobURL.String()] = err
			}
		}

		// Set marker to the next segment
		marker = listBlob.NextMarker
	}

	return errResult

	// blobList, err := containerURL.ListBlobsFlatSegment(ctx, azblob.Marker{}, azblob.ListBlobsSegmentOptions{})
	// if err != nil {

	// }

	// for _, blob := range blobList.Segment.BlobItems {
	// 	blobURL := containerURL.NewBlockBlobURL(blob.Name)
	// 	_, err := blobURL.Delete(ctx, azblob.DeleteSnapshotsOptionNone, azblob.BlobAccessConditions{})
	// 	if err != nil {
	// 		errResult[blobURL.String()] = err
	// 	}
	// }

}
