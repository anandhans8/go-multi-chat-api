# go-multi-chat-api Project Documentation

## Overview

This project is a Go-based microservices application that implements a messaging system with multiple provider support. It follows clean architecture principles to ensure separation of concerns, testability, and maintainability.

## Architecture

The project follows the Clean Architecture pattern, which consists of the following layers:

1. **Domain Layer**: Contains the business entities and interfaces that define the core business rules.
2. **Application Layer**: Contains the use cases that orchestrate the flow of data to and from the entities.
3. **Infrastructure Layer**: Contains the implementation details, such as databases, external services, and frameworks.
4. **Interface Layer**: Contains the controllers, presenters, and gateways that handle the communication with the outside world.

### Clean Architecture Benefits

- **Independence of Frameworks**: The business logic doesn't depend on the existence of any framework.
- **Testability**: The business rules can be tested without the UI, database, web server, or any external element.
- **Independence of UI**: The UI can change easily, without changing the rest of the system.
- **Independence of Database**: The business rules are not bound to the database.
- **Independence of any external agency**: The business rules don't know anything about the outside world.

## Project Structure

```
go-multi-chat-api/
├── docs/                  # Documentation
├── src/                   # Source code
│   ├── application/       # Application layer (use cases)
│   ├── domain/            # Domain layer (entities, interfaces)
│   └── infrastructure/    # Infrastructure layer (implementations)
│       ├── alerting/      # Alerting system
│       ├── di/            # Dependency injection
│       ├── logger/        # Logging system
│       ├── messaging/     # Messaging system
│       ├── repository/    # Repository implementations
│       ├── rest/          # REST API controllers
│       ├── security/      # Security services
│       └── utils/         # Utility functions
├── main.go                # Entry point
├── config.yaml            # Configuration file
└── docker-compose.yml     # Docker Compose configuration
```

## Modules

The project is organized into several modules, each with its own responsibility:

1. **Authentication**: Handles user authentication and authorization.
2. **Messaging**: Manages message sending through different providers.
3. **Signal**: Integrates with the Signal messaging service.
4. **User Management**: Manages user accounts and profiles.

For detailed documentation on each module, please refer to the respective documentation files in this directory.

## Getting Started

To get started with the project, please refer to the main README.md file in the root directory.
