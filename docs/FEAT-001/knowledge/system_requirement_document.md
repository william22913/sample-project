# System Requirement Document (SRD): Teacher Management Module

**Document Metadata**
* **Project Name:** School Database Management System (SDMS)
* **Module:** Teacher Management (CRUD)
* **Author:** Lead Business Analyst
* **Date:** September 28, 2026
* **Version:** 1.0

---

## 1. Executive Summary
The School Database Management System (SDMS) is designed to streamline academic administrative processes, including teacher record management, student course enrollment, class scheduling, and grade/score entry. 

This document defines the detailed business requirements, functional rules, and process flows for the **Teacher Management Module (Teacher CRUD)**. Managing teacher profiles accurately is a critical prerequisite to downstream workflows such as assigning instructors to subjects, scheduling classes, and enabling grade submission.

---

## 2. Business Requirements & Objectives

### 2.1 Business Objectives
* **Centralize Educator Data:** Maintain a single source of truth for all active and historical teacher profiles.
* **Enable Downstream Workflows:** Provide clean data inputs necessary for class allocation and subject grade entry.
* **Auditability & Security:** Ensure only authorized school administrators can modify teacher personnel data while maintaining audit trails for changes.

### 2.2 Scope
* **In-Scope:**
  * Creating new teacher profiles with personal, professional, and contact information.
  * Searching, filtering, and viewing teacher profiles.
  * Updating existing teacher records and assigning/unassigning taught subjects.
  * Soft-deleting/deactivating teacher records to maintain historical integrity.
* **Out-of-Scope (for this phase):**
  * Student registration and enrollment workflows.
  * Interface for teachers entering student scores (handled in the Grading Module).
  * Payroll and salary management.

---

## 3. User Roles & Permissions

| Role | Operations Allowed | Description |
| :--- | :--- | :--- |
| **School Administrator** | Create, Read, Update, Deactivate | Full management access to all teacher records and assignments. |
| **Teacher** | Read (Self-profile only), Update (Limited) | Can view their own profile and update contact details (phone, email). |
| **System/Database** | Soft Delete, Audit Logging | System operations for compliance and integrity. |

---

## 4. Functional Specifications (CRUD Operations)

### 4.1 Create Teacher (`CREATE`)
* **Goal:** Register a new teacher into the system.
* **Inputs/Fields Required:**
  * `Teacher_ID` *(System-generated, Unique, e.g., `TCH-2026-001`)*
  * `First_Name` *(Text, Required, 2–50 chars)*
  * `Last_Name` *(Text, Required, 2–50 chars)*
  * `Email` *(Email format, Required, Unique)*
  * `Phone_Number` *(Numeric/Text, Required)*
  * `Primary_Subject` *(Dropdown/Selection, Required)*
  * `Employment_Status` *(Enum: `Active`, `On Leave`, `Terminated`; Default: `Active`)*
  * `Hire_Date` *(Date, Required)*
* **Business Rules:**
  * Email address must be unique across the entire database.
  * System automatically sends an onboarding account setup link upon creation.

### 4.2 Read / View Teacher (`READ`)
* **Goal:** Search, display, and retrieve teacher records.
* **Capabilities:**
  * Search by `Name`, `Teacher_ID`, or `Subject`.
  * Filter list by `Employment_Status` (`Active` vs. `Inactive`).
  * Detail View shows: Personal Information, Assigned Classes, Taught Subjects, and Active Student Lists.
* **Business Rules:**
  * Pagination default: 20 records per page.

### 4.3 Update Teacher (`UPDATE`)
* **Goal:** Modify existing profile details or subject/class associations.
* **Editable Fields:** Contact details, assigned subjects, employment status.
* **Restricted Fields:** `Teacher_ID` and historical `Hire_Date` cannot be modified without audit approval.
* **Business Rules:**
  * If status changes to `Inactive` or `On Leave`, the system prompts to reassign all currently assigned classes to a substitute teacher.

### 4.4 Delete / Deactivate Teacher (`DELETE`)
* **Goal:** Remove an instructor from active system participation.
* **Business Rules:**
  * **Hard Delete is Strict Violation:** Hard deletion is disabled to preserve student historical grade logs and attendance records.
  * **Soft Delete Strategy:** Teacher record `Status` is toggled to `Inactive`.
  * Deactivated accounts lose system access immediately and are removed from active subject selection dropdowns.

---

## 5. Logical Data Model Entity Specification

### Entity: `TEACHER`
| Attribute | Data Type | Key Type | Nullable | Description |
| :--- | :--- | :--- | :--- | :--- |
| `teacher_id` | INT / VARCHAR(20) | Primary Key | No | Unique identifier |
| `first_name` | VARCHAR(50) | - | No | Teacher's first name |
| `last_name` | VARCHAR(50) | - | No | Teacher's last name |
| `email` | VARCHAR(100) | Unique | No | Official school email |
| `phone` | VARCHAR(20) | - | Yes | Contact number |
| `hire_date` | DATE | - | No | Date hired |
| `status` | ENUM | - | No | `ACTIVE`, `INACTIVE`, `ON_LEAVE` |
| `created_at` | TIMESTAMP | - | No | Record creation timestamp |
| `updated_at` | TIMESTAMP | - | No | Last update timestamp |

---

## 6. Business Process Flow (Use Case Scenario)

```
[Administrator] 
       │
       ▼
 ┌──────────────────────────┐
 │ Open Teacher Management  │
 └─────────────┬────────────┘
               │
      ┌────────┴────────┬───────────────────┬──────────────────┐
      ▼                 ▼                   ▼                  ▼
┌───────────┐     ┌───────────┐       ┌───────────┐      ┌───────────┐
│ Create    │     │ Read/View │       │ Update    │      │ Deactivate│
│ Teacher   │     │ Profiles  │       │ Details   │      │ Profile   │
└─────┬─────┘     └─────┬─────┘       └─────┬─────┘      └─────┬─────┘
      │                 │                   │                  │
      ▼                 ▼                   ▼                  ▼
┌───────────┐     ┌───────────┐       ┌───────────┐      ┌───────────┐
│ Validate  │     │ Display   │       │ Reassign  │      │ Soft      │
│ Email &   │     │ Paginated │       │ Classes   │      │ Delete &  │
│ Fields    │     │ List      │       │ if Status │      │ Revoke    │
└─────┬─────┘     └─────┬─────┘       │ Changes   │      │ Access    │
      │                               └─────┬─────┘      └───────────┘
      ▼                                     │
┌───────────┐                               ▼
│ Save &    │                         ┌───────────┐
│ Send Setup│                         │ Save Log  │
│ Email     │                         └───────────┘
└───────────┘
```

---

## 7. Assumptions & Risk Analysis

* **Assumptions:**
  * System User Authentication (Identity Provider) is already operational or will integrate via OAuth/SAML.
  * A teacher can teach multiple subjects across multiple classes (Many-to-Many relationship).
* **Risks & Mitigations:**
  * **Risk:** Deleting an active teacher mid-semester breaks grade reporting dependencies.
    * *Mitigation:* Enforce system validation checking for active classes before permitting soft deletion.
  * **Risk:** Duplicate profiles created under different email addresses.
    * *Mitigation:* Enforce strict uniqueness checks on government ID / email address attributes during registration.