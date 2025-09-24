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

The BaseServer component provides the following out-of-the-box:

* **Standard HTTP Server Lifecycle**: A blocking Start() method and a graceful Shutdown(ctx) method.
* **Observability Endpoints**:
    * GET /healthz: A liveness probe that always returns 200 OK to signal the service is running.
    * GET /readyz: A readiness probe that returns 200 OK only after the service explicitly signals it's ready via the SetReady(true) method.
    * GET /metrics: Exposes application metrics in the Prometheus format.
* **Dynamic Port Allocation**: Supports using port :0 for automatic port assignment during tests.
* **JSON Response Helpers**: A simple response package for sending standardized JSON responses and errors.

## **Usage Example**

### **1\. Embedding the Base Server**

Embed the microservice.BaseServer struct into your service's main wrapper.

// In your service's main package (e.g., /keyservice/service.go)  
````go
package keyservice

import (  
    "net/http"  
    "github.com/rs/zerolog"

    // Import the base library  
    "github.com/illmade-knight/go-microservice-base/pkg/microservice"  
    "github.com/illmade-knight/go-microservice-base/pkg/response"  
)

type Wrapper struct {  
    *microservice.BaseServer  
    // ... other dependencies like database clients, etc.  
}

func New(cfg *Config, logger zerolog.Logger /*, ...other deps \*/) *Wrapper {  
    baseServer := microservice.NewBaseServer(logger, cfg.HTTPListenAddr)

    // Get the mux and register your service-specific API handlers  
    mux := baseServer.Mux()  
    // mux.Handle("POST /api/v1/...", myApiHandler)  
      
    return &Wrapper{ BaseServer: baseServer }  
}

````

### **2\. Signaling Readiness**

In your main.go, after all dependencies are successfully initialized, you must signal that the service is ready.

// In your service's main executable (e.g., cmd/keyservice/main.go)  
````go
func main() {  
// ... load config, create logger, init database client ...

    service := keyservice.New(cfg, logger, dbClient)  
      
    // After all dependencies are ready, mark the service as ready to serve traffic.  
    service.SetReady(true)

    // ... start the server and handle graceful shutdown ...  
}
````

### **3\. Using Standardized JSON Errors**

In your API handlers, use the response package to send consistent error messages.

// In your service's API handlers (e.g., internal/api/handlers.go)  
````go
import "github.com/illmade-knight/go-microservice-base/pkg/response)"

func (a *API) GetThingHandler(w http.ResponseWriter, r *http.Request) {  
    thing, err := a.Store.GetThing("some-id")  
    if err != nil {  
        // Instead of http.Error(w, "...", 500\)  
        response.WriteJSONError(w, http.StatusNotFound, "the requested thing could not be found")  
        return  
    }

    response.WriteJSON(w, http.StatusOK, thing)  
}
````
## **Development Plan**

* \[x\] **Phase 1: Foundation**: Basic BaseServer with Start/Shutdown and /healthz.
* \[x\] **Phase 2: Observability**: Added /readyz probe and /metrics endpoint.
* \[x\] **Phase 3: Helper Utilities**: Added response package for standardized JSON errors.