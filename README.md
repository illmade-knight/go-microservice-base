# **Go Microservice Base**

This repository provides a lightweight, foundational library for building consistent and robust Go microservices. It solves common boilerplate problems related to the HTTP server lifecycle, graceful shutdowns, and basic observability, allowing developers to focus on business logic.

## **Core Ideology**

The guiding principle of this library is **simplicity and focus**. It is intentionally not a framework. It provides a minimal set of tools to enforce consistency across a fleet of microservices without imposing rigid constraints.

### **The "No Bloat" Rule**

This library **must remain lean**. Its purpose is to handle universal concerns that apply to nearly all web services. It should never contain business logic or features specific to a single service or a small subset of services.

Any proposal to add a new feature must answer "Yes" to the following question:

"Will at least 80% of our microservices need this exact functionality?"

If the answer is "No," the feature belongs in the service's own codebase, not here.

## **Core Features**

### **1\. Standard Server Lifecycle**

The BaseServer component provides the following out-of-the-box:

* **Standard HTTP Server Lifecycle**: A blocking Start() method and a graceful Shutdown(ctx) method.
* **Observability Endpoints**:
    * GET /healthz: A liveness probe that always returns 200 OK to signal the service is running.
    * GET /readyz: A readiness probe that returns 200 OK only after the service explicitly signals it's ready via the SetReady(true) method.
    * GET /metrics: Exposes application metrics in the Prometheus format.
* **Dynamic Port Allocation**: Supports using port :0 for automatic port assignment during tests.

### **2\. Secure Authentication Middleware (JWT)**

The library provides a secure, production-ready middleware for validating JSON Web Tokens (JWTs).

* **Asymmetric RS256 Validation**: The NewJWKSAuthMiddleware is the recommended middleware for all new services. It validates tokens using the industry-standard RS256 algorithm by fetching public keys from a standard JWKS endpoint. This is a highly secure pattern that eliminates the need for shared secrets between services.
* **Automatic Key Caching & Rotation**: The middleware automatically caches the fetched public keys and refreshes them periodically, ensuring high performance and seamless key rotation.
* **Legacy Support (DEPRECATED)**: For backward compatibility, the NewLegacySharedSecretAuthMiddleware is available, but it uses the less secure shared-secret (HS256) pattern and should not be used for new development.

### **3\. Standardized JSON Responses**

* **JSON Response Helpers**: A simple response package for sending standardized JSON payloads and errors ({"error": "message"}), ensuring a consistent API experience for clients.

## **Usage Example**

### **1\. Embedding the Base Server**

Embed the microservice.BaseServer struct into your service's main wrapper.

// In your service's main package (e.g., /keyservice/service.go)  
package keyservice

import (  
"net/http"  
"\[github.com/rs/zerolog\](https://github.com/rs/zerolog)"

    // Import the base library  
    "\[github.com/illmade-knight/go-microservice-base/pkg/microservice\](https://github.com/illmade-knight/go-microservice-base/pkg/microservice)"  
)

type Wrapper struct {  
\*microservice.BaseServer  
// ... other dependencies like database clients, etc.  
}

func New(cfg \*Config, logger zerolog.Logger /\*, ...other deps \*/) \*Wrapper {  
baseServer := microservice.NewBaseServer(logger, cfg.HTTPListenAddr)

    // ... register your handlers on baseServer.Mux() ...  
        
    return \&Wrapper{ BaseServer: baseServer }  
}  
