# Service Layer Guide

This document provides a technical guide to the Service Layer within the Suar backend application. The Service Layer encapsulates the core business logic, acting as an intermediary between the Handler Layer and external resources or data sources. Its primary responsibility is to perform the actual operations requested by the handlers, such as making HTTP requests to external services, processing data, and handling business-specific validations.

## 1. Purpose and Responsibilities

The main responsibilities of the Service Layer include:

-   **Business Logic Execution**: Implementing the core functionality of the application, independent of the transport mechanism (HTTP, gRPC, etc.).
-   **External Communication**: Interacting with external APIs, databases, or other services.
-   **Data Transformation**: Converting data between different formats (e.g., DTOs from handlers to internal models, and vice-versa).
-   **Complex Validation**: Performing business-specific validations that go beyond basic syntax checks (e.g., checking if a URL is allowed based on internal policies).
-   **Error Handling**: Catching and translating errors from external dependencies into application-specific errors.
-   **Security Concerns**: Implementing security measures like SSRF protection.

The Service Layer should be designed to be reusable and testable, without direct dependencies on HTTP-specific constructs.

## 2. Key Components

### `http_proxy_service.go`

This file contains the core logic for proxying HTTP requests.

-   **Constants**:
    -   `maxResponseBodySize`: Defines the maximum size of the response body that will be read (10 MB). This prevents memory exhaustion from very large responses.
    -   `defaultRequestTimeout`: Default timeout for outbound HTTP requests (30 seconds).
    -   `maxRequestTimeout`: Maximum allowed timeout for outbound HTTP requests (90 seconds).
    -   `allowedMethods`: A map of HTTP methods that the proxy service supports.
    -   `blockedHeaders`: A map of HTTP headers that will be stripped from the outbound request for security reasons (e.g., `Authorization`, `Cookie`, `X-Forwarded-For`).

-   **`OutboundRequest struct`**:
    ```go
    type OutboundRequest struct {
        Method  string
        URL     *url.URL
        Headers http.Header
        Body    []byte
        Timeout time.Duration
    }
    ```
    -   An internal representation of the HTTP request to be made to the target server. This struct is derived from `model.DTORequest` after internal validation and transformation.

-   **`isPrivateIP(ip net.IP) bool`**:
    -   A helper function to check if a given IP address belongs to a private (RFC1918) or loopback range. This is a crucial component of SSRF (Server-Side Request Forgery) protection. It prevents the backend from being used to access internal network resources.

-   **`newOutboundRequest(dto *model.DTORequest) (*OutboundRequest, error)`**:
    -   This function is responsible for converting the incoming `model.DTORequest` into an `OutboundRequest` and performing critical validations.
    -   **HTTP Method Validation**: Checks if the `dto.Method` is one of the `allowedMethods`.
    -   **URL Parsing and Scheme Validation**: Parses the `dto.URL` and ensures it uses `http` or `https` schemes.
    -   **SSRF Protection**:
        -   Performs a DNS lookup (`net.LookupIP`) on the hostname of the target URL.
        -   Iterates through the resolved IP addresses and uses `isPrivateIP` to determine if any of them are private. If a private IP is detected, it returns an `ErrInvalidInput` error, preventing the request.
    -   **Timeout Validation**: Ensures the requested `dto.Timeout` is within the `0` (default) to `maxRequestTimeout` range.
    -   **Header Filtering**: Iterates through `dto.Headers` and removes any headers present in the `blockedHeaders` list before creating the `OutboundRequest`.

-   **`HTTPProxyService struct`**:
    ```go
    type HTTPProxyService struct {
        httpClient *http.Client
    }
    ```
    -   Holds an instance of `*http.Client`, which is used to make the actual outbound HTTP requests.

-   **`NewHTTPProxyService() *HTTPProxyService`**:
    -   Constructor for `HTTPProxyService`.
    -   Initializes `http.Client` with a custom `http.Transport`. This transport is configured with:
        -   `MaxIdleConns`: Maximum idle (keep-alive) connections across all hosts.
        -   `IdleConnTimeout`: Amount of time an idle (keep-alive) connection will remain in the pool before closing itself.
        -   `TLSHandshakeTimeout`: Timeout for TLS handshake.
        -   `ExpectContinueTimeout`: Timeout for waiting for a server's first response headers after sending a request with `Expect: 100-continue`.
        -   These settings are crucial for performance and resource management in a proxy service.

-   **`ProcessRequest(ctx context.Context, dto *model.DTORequest) (*model.DTOResponse, error)`**:
    -   This is the public method exposed to the Handler Layer.
    -   It first calls `newOutboundRequest` to validate and transform the `model.DTORequest`.
    -   If `newOutboundRequest` returns an error, it's propagated directly.
    -   If successful, it calls the internal `Execute` method to perform the actual HTTP request.

-   **`Execute(ctx context.Context, outboundRequest *OutboundRequest) (*model.DTOResponse, error)`**:
    -   Performs the actual HTTP request to the target server.
    -   **Context with Timeout**: Creates a new `context.WithTimeout` derived from the incoming `ctx` (from the handler) and the `outboundRequest.Timeout`. This ensures that the HTTP request respects the specified timeout and can be cancelled if the original request context is cancelled.
    -   **`http.NewRequestWithContext`**: Creates the `*http.Request` object using the prepared context, method, URL, and body.
    -   **`s.httpClient.Do(httpRequest)`**: Executes the HTTP request.
    -   **Error Handling**:
        -   Specifically checks if the error is `context.DeadlineExceeded` (which indicates a timeout) and wraps it with `ErrRequestTimeout`.
        -   Other network errors are returned as generic errors.
    -   **Response Conversion**: Calls `httpResponseToDTOResponse` to convert the `*http.Response` into a `model.DTOResponse`.

-   **`httpResponseToDTOResponse(resp *http.Response, duration time.Duration, timestamp time.Time) (*model.DTOResponse, error)`**:
    -   A helper function to convert a standard `*http.Response` into the application's `model.DTOResponse` format.
    -   Reads the response body using `io.LimitedReader` to prevent reading excessively large bodies (up to `maxResponseBodySize`).
    -   Populates `StatusCode`, `Duration`, `Timestamp`, `Headers`, `Size`, and `Body`.
    -   If the response body was truncated, it sets an `Error` message in the `DTOResponse`.

### `errors.go`

This file defines custom error types used within the Service Layer.

-   **`ErrInvalidInput`**: Used for errors related to invalid or malformed input data that prevents the service from processing the request (e.g., invalid URL, unsupported method).
-   **`ErrRequestTimeout`**: Used when an outbound HTTP request times out.

These custom error types allow for more precise error handling and mapping in the Handler Layer.

## 3. Data Flow (Service Layer)

1.  **Request from Handler**: The `ProcessRequest` method is called by the Handler Layer with a `context.Context` and `model.DTORequest`.
2.  **Input Validation and Transformation**: `newOutboundRequest` validates the `DTORequest` (method, URL scheme, timeout) and performs critical security checks (SSRF protection). It transforms the `DTORequest` into an internal `OutboundRequest` model.
3.  **HTTP Client Execution**: The `Execute` method is called with the `OutboundRequest`.
    -   A new `context.Context` with a specific timeout is created for the outbound request.
    -   An `http.Request` is constructed.
    -   The `http.Client` sends the request to the target URL.
4.  **Response Processing**: The `httpResponseToDTOResponse` function reads the response from the target server, limits the body size, and converts it into a `model.DTOResponse`.
5.  **Error Handling**: If any errors occur during validation, request execution, or response reading, appropriate custom errors (`ErrInvalidInput`, `ErrRequestTimeout`) or generic errors are returned.
6.  **Response to Handler**: The `model.DTOResponse` (or an error) is returned to the Handler Layer.

## 4. Security Considerations

The Service Layer is crucial for implementing security measures, especially for a proxy service:

-   **SSRF Protection**: The `isPrivateIP` and the logic within `newOutboundRequest` prevent the application from being used to scan or access internal network resources. This is a fundamental security control for any proxy.
-   **Header Filtering**: Stripping sensitive headers like `Authorization` or `Cookie` from outbound requests prevents accidental leakage of credentials to unintended third parties.
-   **Response Body Size Limiting**: Prevents denial-of-service attacks or memory exhaustion by limiting the amount of data read from external responses.
-   **Timeout Enforcement**: Prevents long-running or stuck requests from consuming server resources indefinitely.

## 5. Testability Considerations

-   **Clear Separation of Concerns**: The service layer is independent of HTTP transport details, making it easier to test its business logic in isolation.
-   **Dependency Injection**: The `http.Client` is injected (via the constructor), allowing for easy mocking of HTTP responses during testing.
-   **Custom Error Types**: Using custom error types (`errors.Is`) simplifies testing error paths and ensures consistent error handling.

## 6. Future Enhancements

-   **Advanced SSRF Protection**: Implement more sophisticated SSRF protection, potentially using a whitelist of allowed domains or IP ranges.
-   **Request/Response Logging**: Add more detailed logging of outbound requests and inbound responses for debugging and auditing purposes (ensure sensitive data is not logged).
-   **Circuit Breaker**: Implement a circuit breaker pattern to prevent cascading failures when an external service is unhealthy.
-   **Retry Mechanism**: Add a configurable retry mechanism for transient network errors.
-   **Metrics and Monitoring**: Integrate with a metrics system (e.g., Prometheus) to collect data on request durations, error rates, etc.
