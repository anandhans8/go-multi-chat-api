# Clean Architecture Documentation

## Overview

This project follows the Clean Architecture pattern, which is a software design philosophy that separates the concerns of a software system into distinct layers. This separation makes the system more maintainable, testable, and adaptable to change.

## Layers

The Clean Architecture consists of the following layers, from the innermost to the outermost:

1. **Domain Layer**: Contains the business entities and interfaces that define the core business rules.
2. **Application Layer**: Contains the use cases that orchestrate the flow of data to and from the entities.
3. **Infrastructure Layer**: Contains the implementation details, such as databases, external services, and frameworks.
4. **Interface Layer**: Contains the controllers, presenters, and gateways that handle the communication with the outside world.

## Dependency Rule

The fundamental rule of Clean Architecture is the Dependency Rule:

> Source code dependencies can only point inwards.

This means that:

- The Domain Layer doesn't depend on any other layer.
- The Application Layer depends only on the Domain Layer.
- The Infrastructure Layer depends on the Domain and Application Layers.
- The Interface Layer depends on the Domain, Application, and Infrastructure Layers.

## Implementation in This Project

In this project, the Clean Architecture is implemented as follows:

### Domain Layer

The Domain Layer is located in the `src/domain` directory and contains:

- Business entities (`User`, `Provider`, etc.)
- Repository interfaces
- Service interfaces

### Application Layer

The Application Layer is located in the `src/application` directory and contains:

- Use cases (`AuthUseCase`, `MessageUseCase`, etc.)
- DTOs (Data Transfer Objects)

### Infrastructure Layer

The Infrastructure Layer is located in the `src/infrastructure` directory and contains:

- Repository implementations
- External service integrations
- Frameworks and libraries

### Interface Layer

The Interface Layer is spread across the `src/infrastructure/rest` directory and contains:

- Controllers
- Middleware
- Routes

## Flow of Control

The flow of control in a Clean Architecture system follows these steps:

1. The Interface Layer receives a request from the outside world.
2. The Interface Layer converts the request into a format suitable for the Application Layer.
3. The Application Layer processes the request using the business rules defined in the Domain Layer.
4. The Application Layer returns the result to the Interface Layer.
5. The Interface Layer converts the result into a format suitable for the outside world and returns it.

## Benefits

The Clean Architecture provides several benefits:

- **Independence of Frameworks**: The business logic doesn't depend on the existence of any framework.
- **Testability**: The business rules can be tested without the UI, database, web server, or any external element.
- **Independence of UI**: The UI can change easily, without changing the rest of the system.
- **Independence of Database**: The business rules are not bound to the database.
- **Independence of any external agency**: The business rules don't know anything about the outside world.

## Diagram

```
+----------------------------------------------------------+
|                     Interface Layer                      |
|                                                          |
|  +--------------------------------------------------+    |
|  |                Infrastructure Layer               |    |
|  |                                                   |    |
|  |  +------------------------------------------+    |    |
|  |  |             Application Layer            |    |    |
|  |  |                                          |    |    |
|  |  |  +----------------------------------+    |    |    |
|  |  |  |           Domain Layer           |    |    |    |
|  |  |  |                                  |    |    |    |
|  |  |  |  +-------------------------+     |    |    |    |
|  |  |  |  |      Entities          |     |    |    |    |
|  |  |  |  +-------------------------+     |    |    |    |
|  |  |  |                                  |    |    |    |
|  |  |  |  +-------------------------+     |    |    |    |
|  |  |  |  |      Interfaces         |     |    |    |    |
|  |  |  |  +-------------------------+     |    |    |    |
|  |  |  |                                  |    |    |    |
|  |  |  +----------------------------------+    |    |    |
|  |  |                                          |    |    |
|  |  |  +----------------------------------+    |    |    |
|  |  |  |         Use Cases               |    |    |    |
|  |  |  +----------------------------------+    |    |    |
|  |  |                                          |    |    |
|  |  +------------------------------------------+    |    |
|  |                                                   |    |
|  |  +------------------------------------------+    |    |
|  |  |         Repositories                     |    |    |
|  |  +------------------------------------------+    |    |
|  |                                                   |    |
|  |  +------------------------------------------+    |    |
|  |  |         External Services                |    |    |
|  |  +------------------------------------------+    |    |
|  |                                                   |    |
|  +--------------------------------------------------+    |
|                                                          |
|  +--------------------------------------------------+    |
|  |         Controllers                              |    |
|  +--------------------------------------------------+    |
|                                                          |
+----------------------------------------------------------+
```

## Conclusion

The Clean Architecture is a powerful design pattern that helps create maintainable, testable, and adaptable software systems. By following the Dependency Rule and separating concerns into distinct layers, we can create systems that are resilient to change and easy to understand.