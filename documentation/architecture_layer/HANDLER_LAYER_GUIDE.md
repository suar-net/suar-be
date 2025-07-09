# Handler Layer Guide

This document provides a technical guide to the Handler Layer within the Suar backend application. The Handler Layer is responsible for receiving incoming HTTP requests, validating them, and delegating the business logic to the Service Layer. It acts as the entry point for all external interactions with the backend.

## 1. Purpose and Responsibilities

The primary responsibilities of the Handler Layer include:

-   **Request Reception**: Listening for and receiving HTTP requests on defined routes.
-   **Request Parsing**: Extracting data from the incoming request (e.g., JSON body, URL parameters, headers).
-   **Request Validation**: Ensuring that the incoming request data conforms to expected formats and constraints. This is crucial for security and data integrity.
-   **Service Delegation**: Invoking the appropriate methods in the Service Layer to perform the core business logic.
-   **Response Formatting**: Transforming the results from the Service Layer into appropriate HTTP responses (e.g., JSON, status codes).
-   **Error Handling**: Catching errors from the Service Layer and translating them into meaningful HTTP error responses.

The Handler Layer should be kept thin, focusing solely on HTTP concerns and delegating complex operations to the Service Layer.

## 2. Key Components

### `router.go`

This file is responsible for setting up the HTTP router (using `go-chi/chi`) and configuring global middleware.

-   **`SetupRouter(s HTTPProxyService, logger *log.Logger) *chi.Mux`**:
    -   This function initializes and configures the `chi.Mux` router.
    -   It takes `HTTPProxyService` (an interface) and a `*log.Logger` as dependencies, which are then injected into the handlers. This promotes dependency inversion and testability.
    -   **Middleware**:
        -   `middleware.Logger`: Logs details of incoming requests (method, path, latency, status code). Essential for debugging and monitoring.
        -   `middleware.Recoverer`: Catches panics during request processing and recovers gracefully, preventing the server from crashing and returning a `500 Internal Server Error`.
        -   `cors.Handler`: Configures Cross-Origin Resource Sharing (CORS).
            -   `AllowedOrigins: []string{"*"}`: **(Development Setting)** Currently set to allow all origins. **For production, this MUST be restricted to specific frontend domains** (e.g., `https://your-frontend.vercel.app`) to prevent security vulnerabilities.
            -   `AllowedMethods`, `AllowedHeaders`, `ExposedHeaders`, `AllowCredentials`, `MaxAge`: Standard CORS configurations.
    -   **Route Definition**:
        -   `r.Route("/api/v1", func(r chi.Router) { ... })`: Defines a versioned API group.
        -   `r.Mount("/request", httpProxyHandler)`: Mounts the `httpProxyHandler` (which implements `http.Handler`) to the `/request` path within the `/api/v1` group. This means all requests to `/api/v1/request` will be handled by `httpProxyHandler`.

### `http_proxy_handler.go`

This file contains the concrete implementation of the HTTP handler for proxying requests.

-   **`HTTPProxyService interface`**:
    ```go
    type HTTPProxyService interface {
        ProcessRequest(ctx context.Context, dto *model.DTORequest) (*model.DTOResponse, error)
    }
    ```
    -   This interface defines the contract that the `HTTPProxyHandler` expects from the Service Layer. It ensures that the handler is decoupled from the specific implementation of the proxy service, making it easier to swap out services or mock them for testing.
    -   `ProcessRequest`: The core method that the handler calls to delegate the actual HTTP request proxying. It takes a `context.Context` (for timeouts and cancellation) and a `model.DTORequest` (the parsed request payload) and returns a `model.DTOResponse` or an error.

-   **`HTTPProxyHandler struct`**:
    ```go
    type HTTPProxyHandler struct {
        service HTTPProxyService
        logger  *log.Logger
    }
    ```
    -   Holds an instance of the `HTTPProxyService` interface and a `*log.Logger`. These are injected via the constructor.

-   **`NewHTTPProxyHandler(s HTTPProxyService, l *log.Logger) *HTTPProxyHandler`**:
    -   Constructor function for `HTTPProxyHandler`.

-   **`ServeHTTP(w http.ResponseWriter, r *http.Request)`**:
    -   This method implements the `http.Handler` interface, making `HTTPProxyHandler` capable of serving HTTP requests.
    -   **Method Check**: Ensures only `POST` requests are accepted. Other methods receive `405 Method Not Allowed`.
    -   **JSON Decoding**:
        -   `json.NewDecoder(r.Body).Decode(&dto)`: Decodes the incoming JSON request body into a `model.DTORequest` struct.
        -   Handles `json.SyntaxError` or `io.EOF` by returning `400 Bad Request`.
    -   **Request Validation**:
        -   `validate.Struct(&dto)`: Uses the `go-playground/validator` library (initialized in `validator.go`) to validate the `dto` struct based on its `validate` tags.
        -   `ValidationError(err)`: A helper function (from `validator.go`) that formats validation errors into a user-friendly string.
        -   Returns `400 Bad Request` if validation fails.
    -   **Service Call**:
        -   `h.service.ProcessRequest(r.Context(), &dto)`: Calls the `ProcessRequest` method on the injected service. `r.Context()` is passed to propagate request-scoped values like deadlines and cancellation signals.
    -   **Error Handling from Service**:
        -   Checks for specific error types returned by the service (`service.ErrInvalidInput`, `service.ErrRequestTimeout`) and maps them to appropriate HTTP status codes (`400 Bad Request`, `504 Gateway Timeout`).
        -   Generic errors from the service are mapped to `500 Internal Server Error`.
        -   Errors are logged using the injected logger.
    -   **Response Sending**:
        -   `h.respondWithJSON(w, http.StatusOK, dtoResponse)`: If the service call is successful, the `model.DTOResponse` is marshaled to JSON and sent with a `200 OK` status.

-   **Helper Functions (`respondWithError`, `respondWithJSON`)**:
    -   `respondWithError(w http.ResponseWriter, code int, message string)`: A utility to send a JSON error response with a given status code and message.
    -   `respondWithJSON(w http.ResponseWriter, code int, payload interface{})`: A generic utility to marshal any payload to JSON and write it to the `http.ResponseWriter` with the specified status code and `Content-Type: application/json` header. Includes basic error handling for JSON marshaling.

### `validator.go`

This file provides the validation logic for request DTOs.

-   **`var validate = validator.New()`**:
    -   Initializes a singleton instance of `go-playground/validator`. This instance is used throughout the handler layer for validating structs.

-   **`ValidationError(err error) string`**:
    -   This function takes a `validator.ValidationErrors` error (which is the type returned by `validate.Struct` when validation fails) and formats it into a human-readable string.
    -   It iterates through each validation error and provides custom messages for common tags like `required`, `url`, `httpmethod`, `gte`, and `lte`. This improves the clarity of error messages returned to the client.

## 3. Data Flow (Handler Layer)

1.  **Incoming Request**: An HTTP `POST` request arrives at `/api/v1/request`.
2.  **Router**: `chi.Router` (configured in `router.go`) receives the request, applies middleware (logging, recovery, CORS), and dispatches it to `HTTPProxyHandler.ServeHTTP`.
3.  **Handler (`ServeHTTP`)**:
    -   Verifies the request method is `POST`.
    -   Decodes the JSON request body into a `model.DTORequest` struct.
    -   Validates the `model.DTORequest` using `go-playground/validator`. If validation fails, a `400 Bad Request` with a descriptive error message is returned.
    -   Calls `h.service.ProcessRequest(r.Context(), &dto)` to delegate the core logic to the Service Layer.
    -   Receives a `model.DTOResponse` or an `error` from the Service Layer.
    -   If an error is received, it checks the error type and sends an appropriate HTTP error response (`400`, `504`, or `500`).
    -   If successful, it marshals the `model.DTOResponse` to JSON and sends it back to the client with a `200 OK` status.

## 4. Error Handling Strategy

The Handler Layer plays a critical role in translating internal application errors into standardized HTTP error responses.

-   **Validation Errors**: Handled directly within the handler, returning `400 Bad Request` with specific messages.
-   **Service Errors**: The handler checks for known error types (e.g., `service.ErrInvalidInput`, `service.ErrRequestTimeout`) and maps them to appropriate HTTP status codes. This ensures that clients receive meaningful feedback.
-   **Unexpected Errors**: Any other errors from the Service Layer are treated as internal server errors and result in a `500 Internal Server Error` to avoid exposing sensitive internal details.
-   **JSON Marshaling/Unmarshaling Errors**: Handled by returning `400 Bad Request` for decoding issues and `500 Internal Server Error` for encoding issues.

## 5. Testability Considerations

-   **Dependency Injection**: The `HTTPProxyHandler` receives its dependencies (`HTTPProxyService` and `*log.Logger`) via its constructor. This makes it easy to mock these dependencies during unit testing, allowing for isolated testing of the handler's logic without needing a running service or actual logging.
-   **Interface-based Service**: Relying on the `HTTPProxyService` interface rather than a concrete struct further enhances testability and flexibility.

## 6. Future Enhancements

-   **More Granular Error Responses**: For complex applications, consider a standardized error response format (e.g., JSON:API errors) that includes error codes, detailed messages, and possibly links to documentation.
-   **Request Tracing**: Integrate with a distributed tracing system (e.g., OpenTelemetry) to trace requests across multiple services.
-   **Rate Limiting**: Implement rate limiting at the handler level to protect against abuse.
-   **Authentication/Authorization**: Add middleware or handler logic for authenticating and authorizing requests before they reach the core business logic.
