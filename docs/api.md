# API Documentation

## Overview

This document provides information about the REST API endpoints available in the application. The API follows RESTful principles and uses JSON for request and response bodies.

## Authentication and Authorization

### Authentication

Most endpoints require authentication using a JWT token. To authenticate, include the token in the `Authorization` header of your request:

```
Authorization: Bearer <token>
```

You can obtain a token by calling the `/auth/login` endpoint with valid credentials.

### Role-Based Authorization

The API implements role-based access control (RBAC) to restrict access to certain endpoints based on user roles. The following roles are available:

- **admin**: Administrators have the highest level of access and can perform all operations.
- **member**: Regular users with standard permissions.

When a role is required for an endpoint, it will be indicated in the endpoint documentation with:

- **Required Role**: `admin` or `member`

Note that admin users can access all endpoints, including those that require the member role, but not vice versa.

## Endpoints

### Authentication

#### Login

Authenticates a user and returns a JWT token.

- **URL**: `/auth/login`
- **Method**: `POST`
- **Auth Required**: No
- **Request Body**:
  ```json
  {
    "username": "string",
    "password": "string"
  }
  ```
- **Response**:
  ```json
  {
    "token": "string",
    "user": {
      "id": "integer",
      "username": "string",
      "email": "string"
    }
  }
  ```

#### Register

Creates a new user account.

- **URL**: `/auth/register`
- **Method**: `POST`
- **Auth Required**: No
- **Request Body**:
  ```json
  {
    "username": "string",
    "email": "string",
    "password": "string"
  }
  ```
- **Response**:
  ```json
  {
    "id": "integer",
    "username": "string",
    "email": "string"
  }
  ```

### User Management

#### Get All Users

Returns a list of all users.

- **URL**: `/users`
- **Method**: `GET`
- **Auth Required**: Yes
- **Required Role**: `admin`
- **Response**:
  ```json
  [
    {
      "id": "integer",
      "username": "string",
      "email": "string"
    }
  ]
  ```

#### Get User by ID

Returns a specific user by ID.

- **URL**: `/users/:id`
- **Method**: `GET`
- **Auth Required**: Yes
- **URL Parameters**: `id=[integer]`
- **Response**:
  ```json
  {
    "id": "integer",
    "username": "string",
    "email": "string"
  }
  ```

#### Create User

Creates a new user.

- **URL**: `/users`
- **Method**: `POST`
- **Auth Required**: Yes
- **Required Role**: `admin`
- **Request Body**:
  ```json
  {
    "username": "string",
    "email": "string",
    "password": "string"
  }
  ```
- **Response**:
  ```json
  {
    "id": "integer",
    "username": "string",
    "email": "string"
  }
  ```

#### Update User

Updates an existing user.

- **URL**: `/users/:id`
- **Method**: `PUT`
- **Auth Required**: Yes
- **Required Role**: `admin`
- **URL Parameters**: `id=[integer]`
- **Request Body**:
  ```json
  {
    "username": "string",
    "email": "string",
    "password": "string"
  }
  ```
- **Response**:
  ```json
  {
    "id": "integer",
    "username": "string",
    "email": "string"
  }
  ```

#### Delete User

Deletes a user.

- **URL**: `/users/:id`
- **Method**: `DELETE`
- **Auth Required**: Yes
- **Required Role**: `admin`
- **URL Parameters**: `id=[integer]`
- **Response**: `204 No Content`

### Messaging

#### Send Message

Sends a message using the specified provider.

- **URL**: `/send/message`
- **Method**: `POST`
- **Auth Required**: Yes
- **Request Body**:
  ```json
  {
    "type": "string",
    "message": "string",
    "recipients": ["string"],
    "user_id": "integer"
  }
  ```
- **Response**:
  ```json
  {
    "id": "integer",
    "status": "string",
    "message": "string"
  }
  ```

#### Get Message Status

Retrieves the status of a previously sent message.

- **URL**: `/send/message/:id/status`
- **Method**: `GET`
- **Auth Required**: Yes
- **URL Parameters**: `id=[integer]`
- **Response**:
  ```json
  {
    "id": "integer",
    "status": "string",
    "message": "string",
    "recipients": ["string"],
    "error_message": "string",
    "retry_count": "integer",
    "created_at": "string",
    "updated_at": "string"
  }
  ```

### Signal

#### Register Number

Registers a new Signal number.

- **URL**: `/signal/register`
- **Method**: `POST`
- **Auth Required**: Yes
- **Request Body**:
  ```json
  {
    "number": "string",
    "use_voice": "boolean",
    "captcha": "string"
  }
  ```
- **Response**: `200 OK`

#### Verify Registered Number

Verifies a registered Signal number.

- **URL**: `/signal/verify`
- **Method**: `POST`
- **Auth Required**: Yes
- **Request Body**:
  ```json
  {
    "number": "string",
    "token": "string",
    "pin": "string"
  }
  ```
- **Response**: `200 OK`

#### Unregister Number

Unregisters a Signal number.

- **URL**: `/signal/unregister`
- **Method**: `POST`
- **Auth Required**: Yes
- **Request Body**:
  ```json
  {
    "number": "string",
    "delete_account": "boolean",
    "delete_local_data": "boolean"
  }
  ```
- **Response**: `200 OK`

#### Get Accounts

Gets all registered Signal accounts.

- **URL**: `/signal/accounts`
- **Method**: `GET`
- **Auth Required**: Yes
- **Response**:
  ```json
  [
    "string"
  ]
  ```

#### Send Signal Message

Sends a message via Signal.

- **URL**: `/signal/send`
- **Method**: `POST`
- **Auth Required**: Yes
- **Request Body**:
  ```json
  {
    "number": "string",
    "message": "string",
    "recipients": ["string"],
    "attachments": ["string"],
    "is_group": "boolean"
  }
  ```
- **Response**:
  ```json
  {
    "timestamp": "integer",
    "results": [
      {
        "recipient": "string",
        "status": "string"
      }
    ]
  }
  ```

#### Receive Signal Messages

Receives messages via Signal.

- **URL**: `/signal/receive`
- **Method**: `POST`
- **Auth Required**: Yes
- **Request Body**:
  ```json
  {
    "number": "string",
    "timeout": "integer",
    "ignore_attachments": "boolean",
    "ignore_stories": "boolean",
    "max_messages": "integer",
    "send_read_receipts": "boolean"
  }
  ```
- **Response**:
  ```json
  {
    "messages": [
      {
        "envelope": {
          "source": "string",
          "sourceDevice": "integer",
          "relay": "string",
          "timestamp": "integer",
          "isReceipt": "boolean",
          "dataMessage": {
            "timestamp": "integer",
            "message": "string",
            "expiresInSeconds": "integer",
            "attachments": [
              {
                "contentType": "string",
                "filename": "string",
                "id": "string",
                "size": "integer"
              }
            ],
            "groupInfo": {
              "groupId": "string",
              "members": ["string"]
            }
          }
        }
      }
    ]
  }
  ```

## Error Handling

The API uses standard HTTP status codes to indicate the success or failure of a request. In case of an error, the response body will contain an error message:

```json
{
  "error": "string"
}
```

Common error codes:

- `400 Bad Request`: The request was malformed or contained invalid parameters.
- `401 Unauthorized`: Authentication is required or the provided credentials are invalid.
- `403 Forbidden`: The authenticated user does not have permission to access the requested resource.
- `404 Not Found`: The requested resource was not found.
- `500 Internal Server Error`: An unexpected error occurred on the server.
