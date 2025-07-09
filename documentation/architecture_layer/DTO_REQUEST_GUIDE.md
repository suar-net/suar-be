# DTO (Data Transfer Object) Guide

This document provides a technical guide to the Data Transfer Objects (DTOs) used within the Suar backend application. DTOs are simple objects that are used to transfer data between different layers of the application, particularly between the Handler Layer (which receives and sends HTTP requests/responses) and the Service Layer (which processes the business logic).

## 1. Purpose and Responsibilities

The primary responsibilities of DTOs include:

-   **Data Encapsulation**: Grouping related data fields into a single object for easy transfer.
-   **Input/Output Definition**: Clearly defining the expected structure of incoming request payloads and outgoing response payloads.
-   **Validation**: Providing a mechanism for basic syntactic and semantic validation of incoming data before it reaches the core business logic.
-   **Decoupling**: Helping to decouple the API contract from the internal domain models, allowing for changes in one without necessarily affecting the other.

## 2. Key Components

### `api.go`

This file defines the `DTORequest` and `DTOResponse` structs, which are central to the data exchange within the application.

#### `DTORequest` Struct

```go
type DTORequest struct {
	Method  string              `json:"method" validate:"required,httpmethod"`
	URL     string              `json:"url" validate:"required,url"`
	Headers map[string][]string `json:"headers"`
	Body    json.RawMessage     `json:"body,omitempty"`
	Timeout int                 `json:"timeout" validate:"gte=0,lte=90000"` // 0 means default, max 90s
}
```

This struct represents the expected JSON payload for an incoming HTTP request to the `/api/v1/request` endpoint. Each field is annotated with `json` tags for JSON marshaling/unmarshaling and `validate` tags for input validation using `go-playground/validator`.

-   **`Method string`**:
    -   `json:"method"`: Maps to the `method` field in the JSON payload.
    -   `validate:"required,httpmethod"`: 
        -   `required`: Ensures this field must be present and non-empty.
        -   `httpmethod`: Custom validation tag (defined in `internal/handler/validator.go` and checked in `internal/service/http_proxy_service.go`) that verifies the string is a valid HTTP method (e.g., GET, POST, PUT, DELETE).
    -   **Purpose**: Specifies the HTTP method for the outbound request.

-   **`URL string`**:
    -   `json:"url"`: Maps to the `url` field in the JSON payload.
    -   `validate:"required,url"`: 
        -   `required`: Ensures this field must be present and non-empty.
        -   `url`: Validates that the string is a syntactically valid URL.
    -   **Purpose**: Specifies the target URL for the outbound request.

-   **`Headers map[string][]string`**:
    -   `json:"headers"`: Maps to the `headers` field in the JSON payload.
    -   **Purpose**: A map of string keys (header names) to string slices (header values) for the outbound request. Allows for multiple values per header (e.g., `Set-Cookie`).

-   **`Body json.RawMessage`**:
    -   `json:"body,omitempty"`: Maps to the `body` field in the JSON payload. `omitempty` means the field will be omitted from the JSON if it's empty.
    -   **Purpose**: Represents the raw JSON body for the outbound request. Using `json.RawMessage` allows the body to be any valid JSON structure (object, array, string, number, boolean, null) without requiring a predefined Go struct, providing flexibility.

-   **`Timeout int`**:
    -   `json:"timeout"`: Maps to the `timeout` field in the JSON payload.
    -   `validate:"gte=0,lte=90000"`: 
        -   `gte=0`: Ensures the timeout value is greater than or equal to 0.
        -   `lte=90000`: Ensures the timeout value is less than or equal to 90000 (milliseconds, representing 90 seconds).
    -   **Purpose**: Specifies the timeout for the outbound request in milliseconds. A value of `0` typically indicates a default timeout should be used by the service layer.

#### `DTOResponse` Struct

```go
type DTOResponse struct {
	StatusCode int                 `json:"status_code"`
	Duration   time.Duration       `json:"duration"`
	Timestamp  time.Time           `json:"timestamp"`
	Size       int64               `json:"size"`
	Headers    map[string][]string `json:"headers"`
	Body       []byte              `json:"body,omitempty"`
	Error      string              `json:"error,omitempty"`
}
```

This struct represents the JSON payload returned as a response from the `/api/v1/request` endpoint after the proxy operation. It encapsulates the details of the target server's response.

-   **`StatusCode int`**:
    -   `json:"status_code"`: Maps to the `status_code` field in the JSON payload.
    -   **Purpose**: The HTTP status code received from the target server (e.g., 200, 404, 500).

-   **`Duration time.Duration`**:
    -   `json:"duration"`: Maps to the `duration` field in the JSON payload.
    -   **Purpose**: The total time taken for the outbound HTTP request to complete, from initiation to receiving the full response body.

-   **`Timestamp time.Time`**:
    -   `json:"timestamp"`: Maps to the `timestamp` field in the JSON payload.
    -   **Purpose**: The time when the outbound request was initiated by the proxy service.

-   **`Size int64`**:
    -   `json:"size"`: Maps to the `size` field in the JSON payload.
    -   **Purpose**: The size of the response body in bytes.

-   **`Headers map[string][]string`**:
    -   `json:"headers"`: Maps to the `headers` field in the JSON payload.
    -   **Purpose**: A map of string keys (header names) to string slices (header values) received from the target server's response.

-   **`Body []byte`**:
    -   `json:"body,omitempty"`: Maps to the `body` field in the JSON payload. `omitempty` means the field will be omitted from the JSON if it's empty.
    -   **Purpose**: The raw byte content of the response body from the target server. This allows for handling various content types (e.g., JSON, HTML, plain text).

-   **`Error string`**:
    -   `json:"error,omitempty"`: Maps to the `error` field in the JSON payload. `omitempty` means the field will be omitted from the JSON if it's empty.
    -   **Purpose**: Contains an error message if any issue occurred during the proxy request (e.g., network error, timeout, body truncation). This field is populated by the service layer.

## 3. Data Flow (DTOs)

1.  **Frontend to Handler**: The frontend sends an HTTP `POST` request with a JSON body conforming to the `DTORequest` structure.
2.  **Handler Processing**: The Handler Layer (`http_proxy_handler.go`) unmarshals the incoming JSON into a `DTORequest` struct and performs initial validation.
3.  **Handler to Service**: The `DTORequest` struct is passed to the Service Layer (`http_proxy_service.go`) for business logic processing.
4.  **Service Processing**: The Service Layer uses the data from `DTORequest` to construct and execute the outbound HTTP request. It then processes the target server's response and populates a `DTOResponse` struct.
5.  **Service to Handler**: The `DTOResponse` struct (or an error) is returned to the Handler Layer.
6.  **Handler to Frontend**: The Handler Layer marshals the `DTOResponse` struct into a JSON payload and sends it back to the frontend as the HTTP response.

## 4. Validation and Error Handling

-   **`go-playground/validator`**: The `validate` tags on `DTORequest` fields are used by the `go-playground/validator` library in the Handler Layer to perform automatic validation. This ensures that basic structural and format requirements are met before the data is processed further.
-   **Custom Validation**: The `httpmethod` validation tag is an example of how custom validation logic can be integrated, ensuring that only supported HTTP methods are accepted.
-   **Error Reporting**: If validation fails, the Handler Layer uses the `ValidationError` helper (from `internal/handler/validator.go`) to generate user-friendly error messages, which are then returned to the client with a `400 Bad Request` status.

## 5. Future Enhancements

-   **Versioned DTOs**: For larger applications, consider versioning DTOs to manage API evolution gracefully.
-   **More Complex Validation Rules**: Implement more intricate validation rules as business requirements grow, potentially using custom validators or more advanced validation libraries.
-   **Schema Generation**: Automate the generation of API schemas (e.g., OpenAPI/Swagger) from DTO definitions to improve documentation and client integration.
