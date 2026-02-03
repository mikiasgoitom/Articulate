# Articulate: A Go Backend Blog API

Articulate is a robust and scalable backend service for a modern blogging platform, written in Go. It is built with a clean architecture to ensure separation of concerns, maintainability, and testability.

This starter project provides a solid foundation for building a feature-rich blog, including functionalities for user authentication, blog post management, comments, likes, and more.

## Features

- **Clean Architecture:** Separates business logic from infrastructure and presentation layers for maintainability.
- **RESTful API:** A well-defined API for all blog functionalities.
- **User Authentication:** Secure user management with JWT-based authentication and Google OAuth2 integration.
- **MongoDB Integration:** Uses MongoDB as the primary data store, managed via a repository pattern.
- **Redis Caching:** Implemented for performance optimization on frequently accessed data.
- **Live Reload:** Pre-configured with `air` for an efficient and fast development workflow.
- **HTTP Middleware:** Includes CORS for cross-origin requests and Tollbooth for rate limiting.
- **Dependency Injection:** Promotes modularity and simplifies testing by decoupling components.
- **Configuration Management:** Centralized configuration loading from a `.yml` file and environment variables.

## Project Structure

The project follows the principles of Clean Architecture to create a separation of concerns.

```
.
├── cmd/api/          # Main application entry point
├── docs/             # Project documentation
├── internal/
│   ├── domain/         # Core business entities and repository interfaces
│   ├── dto/            # Data Transfer Objects for use cases
│   ├── handler/        # HTTP handlers, routing, and middleware
│   ├── infrastructure/ # External services (DB, cache, etc.) and their implementations
│   └── usecase/        # Business logic and application-specific rules
├── .air.toml         # Configuration for live-reloading with Air
├── go.mod            # Go module definition
└── README.md
```

- **`cmd`**: Contains the `main` package, responsible for initializing and starting the application.
- **`internal/domain`**: The core of the application, containing business entities (`entity`) and interfaces for repositories (`contract`). It has no external dependencies.
- **`internal/usecase`**: Implements the business logic by orchestrating data from repositories and executing application-specific rules.
- **`internal/infrastructure`**: Provides concrete implementations for the interfaces defined in the domain layer. This includes database connections (MongoDB), caching (Redis), external services, etc.
- **`internal/handler`**: Manages the presentation layer (HTTP API). It handles incoming requests, calls the appropriate use cases, and returns responses. It includes routing, middleware, and request/response DTOs.
- **`docs`**: Contains detailed documentation, including API endpoints and collaboration guidelines.

## Getting Started

### Prerequisites

- [Go](https://golang.org/doc/install) (version 1.18 or higher)
- [Docker](https://www.docker.com/get-started) (for running MongoDB and Redis)
- [Air](https://github.com/air-verse/air) (for live reload)

### Installation & Setup

1.  **Clone the repository:**

    ```sh
    git clone https://github.com/mikiasgoitom/Articulate.git
    cd Articulate
    ```

2.  **Install dependencies:**

    ```sh
    go mod tidy
    ```

3.  **Install `air` for live-reloading:**

    ```sh
    go install github.com/air-verse/air@latest
    ```

4.  **Set up environment variables:**
    Create a `.env` file in the root directory by copying the example file:

    ```sh
    cp .env.sample .env
    ```

    Then, fill in the required values in the `.env` file (database credentials, API keys, etc.).

5.  **Run services with Docker (optional):**
    If you have Docker installed, you can easily start MongoDB and Redis instances.

6.  **Run the application:**
    Use `air` to start the server. It will automatically watch for file changes and restart the application.
    ```sh
    air
    ```
    The server will start on the port specified in your configuration (default is `8080`).

## API Endpoints

A detailed description of all available API endpoints, including request and response formats, can be found in the Postman collection:
[docs/api_endpoints.md](docs/api_endpoints.md)

## Configuration

The application is configured through a combination of a `config.yml` file and environment variables. Key configuration variables in the `.env` file include:

- `DB_URI`: MongoDB connection string.
- `REDIS_URL`: Redis connection URL.
- `JWT_SECRET_KEY`: Secret key for signing JWTs.
- `GOOGLE_CLIENT_ID` & `GOOGLE_CLIENT_SECRET`: For Google OAuth2.
- `EMAIL_*`: Credentials for the email service (e.g., Mailtrap).

## Contributing

Contributions are welcome! Please read the [collaboration guidelines](docs/COLLABORATION_GUIDELINES.md) before getting started.
