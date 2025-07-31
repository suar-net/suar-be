# Handler Layer Guide

This document provides a technical guide to the Handler Layer within the Suar backend application. The Handler Layer is responsible for receiving incoming HTTP requests, validating them, and delegating the business logic to the Service Layer. It acts as the entry point for all external interactions with the backend.

## 1. Purpose and Responsibilities

The primary responsibilities of the Handler Layer include:

-   **Request Reception**: Listening for and receiving HTTP requests on defined routes.
-   **Request Parsing**: Extracting data from the incoming request (e.g., JSON body, URL parameters, headers).
-   **Request Validation**: Ensuring that the incoming request data conforms to expected formats and constraints.
-   **Service Delegation**: Invoking the appropriate methods in the Service Layer to perform the core business logic.
-   **Response Formatting**: Transforming the results from the Service Layer into appropriate HTTP responses using standardized helper functions.
-   **Error Handling**: Catching errors and translating them into meaningful HTTP error responses.

The Handler Layer should be kept thin, focusing solely on HTTP concerns and delegating complex operations.

## 2. Key Components

### `router.go`

This file is responsible for setting up the HTTP router (using `go-chi/chi`), configuring global middleware, and mounting all application handlers.

-   **`SetupRouter(...) *chi.Mux`**:
    -   This function initializes the `chi.Mux` router and injects dependencies (services, database connections, loggers) into the various handlers.
    -   **Middleware**: Configures global middleware like `Logger` (for request logging), `Recoverer` (for panic recovery), and `CORS` (for Cross-Origin Resource Sharing).
    -   **Handler Initialization**: It creates instances of all necessary handlers, such as `HTTPProxyHandler` and `HealthHandler`, by calling their respective constructors.
    -   **Route Definition**:
        -   `r.Route("/api/v1", ...)`: Defines a versioned API group.
        -   `r.Mount("/request", httpProxyHandler)`: Mounts the `httpProxyHandler` to handle all requests to `/api/v1/request`.
        -   `r.Get("/healthcheck", healthHandler.Check)`: Mounts the `healthHandler`'s `Check` method to handle `GET` requests to `/api/v1/healthcheck`.

### `http_proxy_handler.go`

This file contains the concrete implementation of the HTTP handler for proxying requests.

-   **`HTTPProxyService interface`**: Defines the contract that the handler expects from the Service Layer, ensuring decoupling for testability.
-   **`HTTPProxyHandler struct`**: Holds its dependencies, primarily an instance of the `HTTPProxyService` interface and a logger, which are injected via the constructor.
-   **`NewHTTPProxyHandler(...)`**: Constructor function for `HTTPProxyHandler`.
-   **`ServeHTTP(w http.ResponseWriter, r *http.Request)`**:
    -   Implements the `http.Handler` interface.
    -   **Logic**:
        1.  Validates the request method (accepts only `POST`).
        2.  Decodes the incoming JSON request body into a `model.DTORequest` struct.
        3.  Validates the `DTORequest` struct using the `go-playground/validator` library.
        4.  Calls `h.service.ProcessRequest(...)` to delegate the core logic to the Service Layer.
        5.  Receives a `model.DTOResponse` or an `error` from the service.
        6.  Uses the package-level helper functions (`respondWithError`, `respondWithJson`) to send the final HTTP response to the client.

### `health_handler.go`

This file contains the handler for the application's health check endpoint.

-   **`HealthHandler struct`**: Holds its dependencies, `*sql.DB` for the database connection and a `*log.Logger`.
-   **`NewHealthHandler(...)`**: Constructor function for `HealthHandler`.
-   **`Check(w http.ResponseWriter, r *http.Request)`**:
    -   The actual handler function for the `GET /api/v1/healthcheck` route.
    -   It performs a `PingContext` to the database to verify that the connection is alive.
    -   It uses the `respondWithError` or `respondWithJson` helpers to return a `200 OK` status if the check is successful, or a `503 Service Unavailable` status if it fails.

### `response.go`

This file provides centralized, general-purpose helper functions for creating standardized HTTP responses. Using these helpers ensures all API responses are consistent.

-   **`respondWithJson(w http.ResponseWriter, code int, payload interface{})`**:
    -   A generic utility to marshal any payload to JSON.
    -   It sets the `Content-Type: application/json` header, writes the HTTP status code, and sends the JSON response.
    -   It includes error handling for JSON marshaling failures.
-   **`respondWithError(w http.ResponseWriter, code int, message string)`**:
    -   A utility to send a standardized JSON error response.
    -   It wraps the error message in a consistent JSON object (`{"error": "message"}`) and uses `respondWithJson` to send it. This ensures all error responses have the same format.

### `validator.go`

This file provides the validation logic for request DTOs.

-   **`var validate = validator.New()`**: Initializes a singleton instance of `go-playground/validator`.
-   **`ValidationError(err error) string`**: A helper function that takes a `validator.ValidationErrors` error and formats it into a human-readable string, improving the clarity of error messages returned to the client.

## 3. Data Flow (Handler Layer)

1.  **Incoming Request**: An HTTP request arrives at a configured endpoint (e.g., `POST /api/v1/request` or `GET /api/v1/healthcheck`).
2.  **Router**: `chi.Router` receives the request, applies middleware, and dispatches it to the appropriate handler (`HTTPProxyHandler` or `HealthHandler`).
3.  **Handler Logic**:
    -   The handler parses and validates the request.
    -   It calls the appropriate service method (for `httpProxyHandler`) or performs its internal logic (like `healthHandler` pinging the DB).
    -   It receives data or an error back from its operation.
4.  **Response Generation**: The handler uses the helper functions from `response.go` (`respondWithJson` or `respondWithError`) to create and send a consistent JSON response to the client.

## 4. Error Handling Strategy

-   **Validation Errors**: Handled directly within the handler, returning `400 Bad Request` with specific messages generated by `validator.go`.
-   **Service Errors**: The handler checks for known error types from the service layer and maps them to appropriate HTTP status codes (`400`, `504`, `500`).
-   **Standardized Responses**: All error responses are funneled through `respondWithError`, ensuring a consistent format.

## 5. Testability Considerations

-   **Dependency Injection**: All handlers receive their dependencies via their constructors. This makes it easy to mock these dependencies during unit testing.
-   **Centralized Helpers**: Logic for validation and response formatting is centralized in helper files (`validator.go`, `response.go`), which can be tested independently.
