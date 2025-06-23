# Watch Party API Service

This is the API service for the Watch Party application, providing authentication and user management functionality.

## Features Implemented

### 1. User Registration
- **Admin Registration**: `POST /api/v1/admin/register`
- **User Registration**: `POST /api/v1/users/register`

### 2. User Authentication
- **Login**: `POST /api/v1/auth/login`
- **Logout**: `POST /api/v1/auth/logout`


## Error Responses

All endpoints return appropriate HTTP status codes and error messages:

- `400 Bad Request`: Invalid request payload
- `401 Unauthorized`: Invalid credentials or missing/invalid token
- `403 Forbidden`: Insufficient permissions
- `409 Conflict`: User already exists (during registration)
- `500 Internal Server Error`: Server error

## Setup and Running

1. **Setup Database**:
   - Create a PostgreSQL database
   - Run the schema from `db/schema.sql`

2. **Environment Variables**:
   - Copy `.env.example` to `.env`
   - Update the database and JWT secret configurations

3. **Build and Run**:
   ```bash
   go build -o ./bin/api-service ./service-api/cmd/main.go
   ./bin/api-service
   ```

## Authentication Flow

1. **Registration**: Users register with email and password
2. **Login**: Users authenticate and receive JWT access and refresh tokens
3. **Protected Requests**: Include `Authorization: Bearer <access-token>` header
4. **Logout**: Invalidate refresh token

## Security Features

- Password hashing using bcrypt
- JWT tokens for stateless authentication
- Refresh token storage and management
- Input validation and sanitization
- SQL injection prevention using parameterized queries

## Architecture
TODO: elaborate further.
The service follows a layered architecture:
- **Controllers**: Handle HTTP requests and responses
- **Services**: Business logic layer
- **Repositories**: Data access layer
- **Models**: Data structures and validation


