# Technical Documentation: QTERA AI-Powered IT Asset Management & Intelligent Assistant Platform

> **Document Version:** 2.0.0  
> **Last Updated:** September 2026  
> **Repository:** `QTERA/pocAI2`  
> **Confidentiality:** Internal Engineering & Technical Specification  

---

## 1. Executive Overview

The **QTERA Asset Management & Intelligent Assistant Platform** is an enterprise-grade solution that pairs IT Asset Lifecycle Management with an Agentic AI Copilot. The system enables organizations to manage physical and digital IT assets through both a modern web application and an intelligent conversational assistant capable of understanding natural language commands, processing audio dictations, parsing attached documents (PDF/OCR), and autonomously executing database operations.

### Key Capabilities

1. **IT Asset Inventory & Lifecycle Management**: Complete CRUD tracking of hardware assets, models, brands, acquisition costs, purchasing dates, and physical locations.
2. **Multi-Tenant Data Segregation**: Strict company-level (`perusahaan`) isolation across web interfaces, backend API routes, and AI tool calling.
3. **Role-Based Access Control (RBAC)**: Fine-grained access control across three user tiers (`admin`, `operator`, and `viewer`).
4. **Multi-turn Reasoning**: Multi-turn reasoning loops that translate natural language queries and attached documentation into structured tool calls over standard MCP channels.
5. **Prompt & Token Optimization**: Automated tool definition minification (via Caveman token compression) to minimize prompt overhead and context window latency on local and cloud LLMs.
6. **Multimodal Input & Document Ingestion**: Text extraction from uploaded PDFs and image OCR via Tesseract, combined with offline Whisper speech-to-text transcription.
7. **Real-Time Communication**: WebSocket-driven conversational channel with persistent history and browser-native Text-to-Speech (TTS) audio playback.

---

## 2. High-Level System Architecture

The platform is designed as a distributed, decoupled multi-service system comprising a frontend single-page application, a core API backend, a dedicated AI/RAG orchestrator, an MCP tool server, offline speech transcription, and a relational PostgreSQL database.

```mermaid
flowchart TB
    subgraph Client_Layer ["Client Layer (Port 5173)"]
        UI["ai-registration-frontend<br/>(React 18 + Vite + Phosphor Icons)"]
    end

    subgraph Core_Backend ["Core Backend Server (Port 3000)"]
        FiberServer["Go Fiber Server<br/>(server/GO/main.go)"]
        WSHub["WebSocket Connection Hub<br/>(/ws/:id)"]
        JWTAuth["JWT Authentication & RBAC Middleware"]
        WhisperSub["Whisper Speech-to-Text Engine<br/>(whisper.cpp + FFmpeg)"]
    end

    subgraph AI_Orchestration ["AI & RAG Orchestrator (Port 8000)"]
        EIAI["eiai_go Server<br/>(eiai_go/main.go)"]
        CallToolsBlock["controller.CallTools()<br/>(Master Multi-Turn Agent & Router)"]
        ExtractContextBlock["controller.extractContext()<br/>(Analytical Sufficiency & Verdict Engine)"]
        Caveman["Caveman Token Minifier<br/>(ShrinkToolCatalog)"]
        ExtractionEngine["Multimodal Extractor<br/>(PDF Parser + Tesseract OCR)"]
    end

    subgraph MCP_Layer ["Model Context Protocol (Stdio)"]
        MCPServer["mcp_server<br/>(mcp_server/server.go)"]
        Tools["Tool Registry:<br/>• get_assets<br/>• add_asset<br/>• update_asset<br/>• delete_asset"]
    end

    subgraph External_LLM ["External LLM Providers"]
        OpenRouter["OpenRouter / LLM Inference Endpoint<br/>• ling-3.0-flash-fin<br/>• liquid/lfm-2.5-2.6b<br/>• Qwen"]
    end

    subgraph Data_Storage ["Data Persistence Layer"]
        Postgres[(PostgreSQL 14+ 'poc_ai'<br/>pgxpool Connection Pool)]
    end

    %% Client Interactions
    UI -->|"HTTP REST Requests (/api/*)"| FiberServer
    UI -->|"WebSocket Realtime (/ws/*)"| WSHub
    UI -->|"Audio Upload (/transcribe, /message/audio)"| WhisperSub

    %% Core Backend Interactions
    FiberServer -->|"Database Operations"| Postgres
    FiberServer -->|"Forward Chat / Attachment (/rag)"| EIAI
    WSHub -.->|"Broadcast Events"| UI

    %% AI Pipeline Interactions
    EIAI -->|"Initialize / Pipe Stdio"| MCPServer
    EIAI -->|"Document Content"| ExtractionEngine
    EIAI -->|"Compress Tool Definitions"| Caveman
    EIAI -->|"Reasoning & Generation"| OpenRouter
    EIAI -->|"Verify Context & Fetch History"| FiberServer

    %% MCP Execution
    MCPServer --> Tools
    Tools -->|"Authenticated REST Calls with Scoped Headers"| FiberServer
```

---

## 3. Component Deep Dive

### 3.1 Core API & WebSocket Server (`server/GO`)

The core server is written in Go utilizing the high-performance [Fiber v2](https://gofiber.io/) web framework. It serves as the primary system of record, authentication provider, and WebSocket communications broker.

* **Entry Point:** [server/GO/main.go](file:///c:/Users/user/OneDrive/Documents/proyek_DH/QTERA/pocAI2/server/GO/main.go)
* **Configuration:** Loaded from [server/GO/.env](file:///c:/Users/user/OneDrive/Documents/proyek_DH/QTERA/pocAI2/server/GO/.env)
* **Default Port:** `3000`
* **Database Driver:** `github.com/jackc/pgx/v5/pgxpool`

#### Core Responsibilities
1. **Authentication & Session Lifecycle:**
   - Issues HMAC-SHA256 signed JSON Web Tokens (JWT) valid for 24 hours.
   - Enforces role verification and multi-tenant company binding during registration and login.
2. **Asset Inventory Management:**
   - Exposes REST endpoints for querying, filtering, sorting, paginating, creating, updating, and deleting IT assets.
   - Restricts all modifications to assets matching the authenticated user's `company` (`perusahaan`).
3. **Real-time WebSocket Hub:**
   - Maintains an in-memory connection registry `wsHub` keyed by `chatID`.
   - Distributes incoming user queries and asynchronous AI assistant replies in real time.
---

### 3.2 AI & RAG Orchestration Engine (`eiai_go`)

The AI engine coordinates agentic tool calling, prompt synthesis, multimodal file extraction, and external LLM interactions.

* **Entry Point:** [eiai_go/main.go](file:///c:/Users/user/OneDrive/Documents/proyek_DH/QTERA/pocAI2/eiai_go/main.go)
* **Controller Directory:** [eiai_go/controller/](file:///c:/Users/user/OneDrive/Documents/proyek_DH/QTERA/pocAI2/eiai_go/controller/)
* **Default Port:** `8000`
* **Communication Protocol:** Stdio MCP Client to `mcp_server`

#### Pipeline Architecture

```mermaid
sequenceDiagram
    autonumber
    participant U as User / Frontend
    participant S as server/GO (:3000)
    participant E as eiai_go (:8000)
    participant LLM as OpenRouter / Model
    participant MCP as mcp_server (Stdio)
    participant DB as PostgreSQL

    U->>S: POST /message (text, model, attachment)
    S->>S: Persist User Message & Broadcast WebSocket
    S->>E: Forward to /rag (query, token, chat_id, file)
    
    opt Multimodal Attachment Present
        E->>E: Extract Text (pdf.GetPlainText / Tesseract OCR)
    end

    E->>E: Fetch User Context (/api/auth/me) & Chat History (/messages/:id)
    E->>E: Compress Tool Catalog (Caveman Minification)
    
    loop Tool Reasoning Loop (Max 3 Iterations)
        E->>LLM: Send System Prompt + History + Context + Tools
        LLM-->>E: Output Reasoning & <tool_call> JSON
        E->>E: Parse Tool Calls & Resolve Identifiers
        E->>MCP: Execute Tool (get_assets / add_asset / etc.)
        MCP->>S: Authenticated REST Call with Company Headers
        S->>DB: SQL Query / Mutation
        DB-->>S: Query Results
        S-->>MCP: JSON Response
        MCP-->>E: Tool Execution Result
        E->>LLM: Evaluate Sufficiency (<verdict status="ENOUGH|NEED_MORE">)
        LLM-->>E: Verdict Received
    end

    E->>LLM: Final Response Synthesis (GenerateResponse)
    LLM-->>E: Natural Language Response
    E-->>S: Return Final Answer
    S->>S: Persist Assistant Message & Broadcast WebSocket
    S-->>U: HTTP 200 OK (Response stream delivered via WebSocket)
```

#### `eiai_go` Architecture & Execution Flowchart

The AI orchestrator is built around two primary functions in [eiai_go/controller/models.go](file:///c:/Users/user/OneDrive/Documents/proyek_DH/QTERA/pocAI2/eiai_go/controller/models.go):
1. **`controller.CallTools()`**: The master multi-turn agent that filters available tools, shrinks schemas with Caveman, invokes OpenRouter models, resolves identifiers, executes MCP tools in parallel, and coordinates the iteration loop.
2. **`controller.extractContext()`**: An analytical sub-agent evaluator that inspects accumulated live tool outputs, performs temporal and mathematical validation, ensures mutations were executed, and emits a structured sufficiency verdict (`<verdict status="ENOUGH|NEED_MORE">`).

```mermaid
flowchart TD
    subgraph Route_Handler ["1. HTTP Entry: main.go (Route /rag)"]
        Req["HTTP Request /rag<br/>(query, role, token, chat_id, attachment)"]
        UserCtx["FetchUserContext & FetchChatHistory"]
        DocExtract{"Has Attachment?"}
        RunExtract["ExtractText (PDF parser / Tesseract OCR)"]
        InvokeCallTools["Invoke controller.CallTools(...)"]

        Req --> UserCtx --> DocExtract
        DocExtract -- Yes --> RunExtract --> InvokeCallTools
        DocExtract -- No --> InvokeCallTools
    end

    subgraph CallTools_Engine ["2. controller.CallTools() — Master Agent & Multi-Turn Router"]
        InitTools["1. filterRoles (RBAC filtering)<br/>2. ShrinkToolCatalog (Caveman Token Minifier)"]
        BuildPrompts["Assemble System Prompt, Temporal Reference & Conversation History"]
        LoopStart["Multi-Turn Reasoning Loop (Iteration 1..maxIterations)"]
        ChatGen["ChatGenerate (OpenRouter LLM API)"]
        ParseCalls["tryParseToolCalls (&lt;tool_call&gt; tags / JSON / text)"]
        ResolveIDs["resolveIdentifiers (Smart Name-to-ID Matching)"]
        CheckCalls{"New Tool Calls<br/>Found?"}
        
        MutationCheck{"Iteration 1 &<br/>isMutationIntent?"}
        ForceMutation["Inject Prompt: 'Please proceed by calling tool'"]
        ExecTools["executeToolsParallel (Stdio MCP Client ➔ mcp_server)"]
        InvokeExtractContext["Invoke controller.extractContext(...)"]

        LoopStart --> ChatGen --> ParseCalls --> ResolveIDs --> CheckCalls
        CheckCalls -- No --> MutationCheck
        MutationCheck -- Yes --> ForceMutation --> ChatGen
        MutationCheck -- No --> JumpGenResponse
        CheckCalls -- Yes --> ExecTools --> InvokeExtractContext
    end

    subgraph ExtractContext_Engine ["3. controller.extractContext() — Analytical Sufficiency Sub-Agent"]
        ExtractPrompt["Build Analytical System Prompt<br/>(Live DB Data + Temporal Context + Company Scope)"]
        EvalLLM["ChatGenerate (LLM Sufficiency Evaluator)"]
        ParseVerdict["Parse &lt;verdict status='ENOUGH|NEED_MORE'&gt;"]
        MutationGuard{"isMutationQuery &&<br/>!hasExecutedMutation?"}
        ForceNeedMore["Override isEnough = false<br/>missing = 'Mutation tool not executed yet'"]
        ReturnVerdict["Return (isEnough, extractedFacts, missingInfo)"]

        InvokeExtractContext --> ExtractPrompt --> EvalLLM --> ParseVerdict --> MutationGuard
        MutationGuard -- Yes --> ForceNeedMore --> ReturnVerdict
        MutationGuard -- No --> ReturnVerdict
    end

    subgraph Evaluation_Branch ["Loop Control & Iterative Feedback"]
        EvalVerdict{"isEnough == true OR<br/>iter &gt;= maxIterations?"}
        FeedbackHistory["Append Feedback to Messages:<br/>• Assistant thought & tool call<br/>• Tool execution results<br/>• Missing information hint"]
    end

    subgraph Response_Engine ["4. controller.GenerateResponse()"]
        JumpGenResponse["GenerateResponse Prompt Synthesis"]
        FinalAnswer["Ground-Truth Bounded Text Answer"]
        FinalReturn(["HTTP 200 Plain Text Response"])

        JumpGenResponse --> FinalAnswer --> FinalReturn
    end

    InvokeCallTools --> InitTools --> BuildPrompts --> LoopStart
    ReturnVerdict --> EvalVerdict
    EvalVerdict -- No --> FeedbackHistory --> ChatGen
    EvalVerdict -- Yes --> JumpGenResponse
```

---

### 3.2.1 Deep Dive: `controller.CallTools()`

* **Source File:** [eiai_go/controller/models.go](file:///c:/Users/user/OneDrive/Documents/proyek_DH/QTERA/pocAI2/eiai_go/controller/models.go#L340-L493)
* **Function Signature:**
  ```go
  func CallTools(
      ctx context.Context,
      client *client.Client,
      mcpTools []mcp.Tool,
      query string,
      role string,
      userContext map[string]any,
      maxIterations int,
      model string,
  ) (string, error)
  ```

#### Architectural Purpose & Workflow:
`CallTools` is the master controller for the entire agentic pipeline. It manages the multi-turn interaction loop between the user query, external LLM, and the MCP tool server:

1. **Role Filtering (`filterRoles`)**:
   Enforces role-based tool availability before constructing system prompts. A `viewer` only sees `get_assets`; an `operator` sees `get_assets`, `add_asset`, and `update_asset`; an `admin` sees all tools including `delete_asset`.
2. **Token Minification (`ShrinkToolCatalog`)**:
   Invokes `caveman-shrink` CLI or `NativeCavemanShrink` (a zero-dependency regex engine) to strip filler words and condense descriptions in the tool definitions by 30–50%, substantially lowering latency and token usage.
3. **Multi-Turn Reasoning Loop (Default: up to 3 iterations)**:
   - Dispatches system instructions, conversational history, attachment context, and the user prompt to OpenRouter via `ChatGenerate`.
   - Employs regex and JSON parsers (`tryParseToolCalls`) to extract structured `<tool_call>` tags from model output.
   - Invokes `resolveIdentifiers` to dynamically resolve human-friendly names to actual database asset IDs (`AST-XXXXXXXX`).
   - Deduplicates calls against `executedSignatures` to prevent infinite tool-calling loops.
4. **Mutation Intent Safeguard**:
   If the user's prompt signals a mutation (`isMutationIntent` detects keywords like *create*, *add*, *update*, *delete*, *hapus*, *ubah*) but the model emits plain text on Iteration 1 without calling a tool, `CallTools` intercepts the output and re-prompts the model:
   > *"Please proceed with the operation by calling the appropriate tool (e.g. `<tool_call>{"name": "add_asset", "arguments": {...}}</tool_call>`)"*
5. **Parallel Execution (`executeToolsParallel`)**:
   Spawns concurrent Go goroutines using `sync.WaitGroup` to execute multiple tool calls simultaneously over the Stdio MCP client.
6. **Sub-Agent Invocation (`extractContext`)**:
   Passes accumulated tool results to `extractContext` on every iteration to verify data completeness.
7. **Final Synthesis (`GenerateResponse`)**:
   Once data sufficiency is confirmed, calls `GenerateResponse` to produce a natural language answer strictly bound to the verified context.

---

### 3.2.2 Deep Dive: `controller.extractContext()`

* **Source File:** [eiai_go/controller/models.go](file:///c:/Users/user/OneDrive/Documents/proyek_DH/QTERA/pocAI2/eiai_go/controller/models.go#L187-L322)
* **Function Signature:**
  ```go
  func extractContext(
      ctx context.Context,
      query string,
      accumulatedResults []ToolExecutionResult,
      userContext map[string]any,
      iteration int,
      maxIterations int,
      model string,
  ) (bool, string, string)
  ```
* **Returns:**
  * `isEnough` (`bool`): Whether the accumulated data is sufficient to answer the user's request.
  * `extractedFacts` (`string`): Synthesized factual data with dates and counts evaluated.
  * `missingInfo` (`string`): Explicit description of what information remains missing.

#### Architectural Purpose & Sufficiency Protocol:
`extractContext` acts as an analytical sub-agent evaluator that verifies ground-truth data completeness before the model answers the user:

1. **Analytical System Prompt & Temporal Grounding**:
   - Injects temporal references via `getTemporalContextPrompt()` (e.g., current ISO timestamp, today's date, start of current week, current month prefix `YYYY-MM`).
   - Injects requesting tenant company scoping (`userContext["company"]`).
2. **Strict Sufficiency Protocol (`<verdict>`)**:
   On the very first line of its response, the LLM evaluator is mandated to return a verdict tag:
   - `<verdict status="ENOUGH"/>`: Live tool data fully satisfies all criteria requested by the user.
   - `<verdict status="NEED_MORE" missing="specific missing detail"/>`: Crucial data is still missing, signalling `CallTools` to continue to the next iteration.
3. **Analytical Calculations**:
   When queries involve counting or aggregations (*"how many laptops are at Warehouse A?"*), `extractContext` forces explicit tallying and verification against the database records.
4. **Strict Mutation Execution Enforcement**:
   Even if the evaluator model attempts to emit `<verdict status="ENOUGH"/>`, `extractContext` executes a programmatic safeguard:
   ```go
   isMutationQuery := isMutationIntent(query)
   hasExecutedMutation := false
   for _, res := range accumulatedResults {
       if res.Tool == "update_asset" || res.Tool == "delete_asset" || res.Tool == "add_asset" {
           hasExecutedMutation = true
           break
       }
   }
   if isMutationQuery && !hasExecutedMutation && iteration < maxIterations {
       isEnough = false
       missingInfo = "The requested asset mutation (update/delete/add) has not been executed yet. Proceed to call the appropriate mutation tool."
   }
   ```
   This prevents the assistant from falsely claiming an asset was updated or deleted when no database mutation took place.

---

#### Supporting Modules within `eiai_go`:
1. **[toolCall.go](file:///c:/Users/user/OneDrive/Documents/proyek_DH/QTERA/pocAI2/eiai_go/controller/toolCall.go)**:
   - **`ROLE_TOOL_PERMISSIONS`**: RBAC permissions mapping.
   - **`ShrinkToolCatalog` & `NativeCavemanShrink`**: Prompt token minifier.
   - **`executeToolsParallel`**: Concurrency coordinator for MCP calls.
2. **[identifier.go](file:///c:/Users/user/OneDrive/Documents/proyek_DH/QTERA/pocAI2/eiai_go/controller/identifier.go)**:
   - **`resolveIdentifiers`**: Analyzes entity references in tool call arguments and maps human names to database asset IDs (`AST-XXXXXXXX`).
3. **[extract.go](file:///c:/Users/user/OneDrive/Documents/proyek_DH/QTERA/pocAI2/eiai_go/controller/extract.go)**:
   - Multimodal document parser for PDF text extraction and image OCR via Tesseract.
4. **[decode_jwt.go](file:///c:/Users/user/OneDrive/Documents/proyek_DH/QTERA/pocAI2/eiai_go/controller/decode_jwt.go)**:
   - Validates tokens, resolves user profile, company, and role contexts, and pulls recent conversation history (`FetchChatHistory`).

---

### 3.3 Model Context Protocol (MCP) Server (`mcp_server`)

The MCP Server implements the open Model Context Protocol standard over Stdio transport using `github.com/mark3labs/mcp-go`.

* **Entry Point:** [mcp_server/server.go](file:///c:/Users/user/OneDrive/Documents/proyek_DH/QTERA/pocAI2/mcp_server/server.go)
* **Transport:** Standard Input/Output (`server.ServeStdio`)
* **Upstream Target:** `http://127.0.0.1:3000/api`

#### Registered MCP Tools:

| Tool Name | Accessible Roles | Description | Primary Parameters |
|:---|:---|:---|:---|
| **`get_assets`** | `admin`, `operator`, `viewer` | Query and filter the asset catalog scoped to the user's company. | `search`, `category`, `brand`, `location`, `sort`, `order`, `pageSize`, `page` |
| **`add_asset`** | `admin`, `operator` | Insert a new asset record directly into the database. | `name` (required), `category` (required), `brand`, `modelType`, `purchaseDate`, `purchasePrice`, `location` |
| **`update_asset`**| `admin`, `operator` | Modify existing asset properties by asset ID. | `id` (required), `name`, `category`, `brand`, `modelType`, `purchaseDate`, `purchasePrice`, `location`, `status` |
| **`delete_asset`**| `admin` only | Delete an asset record by asset ID. Rejects operators and viewers. | `id` (required) |

#### Context Injection & Security
When tools are called by `eiai_go`, internal credentials (`Credentials` struct containing `token`, `user_id`, `company`, and `role`) are injected into the tool arguments. The MCP server extracts these credentials, injects them into HTTP request headers (`Authorization: Bearer <token>`, `X-Company`, `X-Role`, `X-User-ID`), and invokes the core backend API. The core backend enforces that no user or tool can modify or inspect records outside their company boundaries.

---

### 3.4 Web Client (`ai-registration-frontend`)

A modern Single-Page Application (SPA) designed with pure CSS custom properties, Google Fonts Inter, and Phosphor Icons.

* **Entry Point:** [ai-registration-frontend/src/main.jsx](file:///c:/Users/user/OneDrive/Documents/proyek_DH/QTERA/pocAI2/ai-registration-frontend/src/main.jsx)
* **Application Shell:** [ai-registration-frontend/src/App.jsx](file:///c:/Users/user/OneDrive/Documents/proyek_DH/QTERA/pocAI2/ai-registration-frontend/src/App.jsx)
* **Chat Component:** [ai-registration-frontend/src/Chat.jsx](file:///c:/Users/user/OneDrive/Documents/proyek_DH/QTERA/pocAI2/ai-registration-frontend/src/Chat.jsx)
* **Dev Server:** Port `5173` (proxies `/api`, `/ws`, `/message`, `/messages`, `/transcribe` to port `3000`).

#### Key Features:
1. **Authentication & Identity Management:**
   - Integrated Sign In & Sign Up interfaces.
   - Dynamic company selection backed by `/api/perusahaan`.
   - Role badge display (`admin`, `operator`, `viewer`) with color-coded tags.
2. **Asset Management Dashboard:**
   - Full-text search across asset IDs, names, brands, and categories.
   - Multi-select facet filter panels for Category, Brand, and Physical Location.
   - Dynamic column visibility toggles (Asset ID and Name are fixed; others customizable).
   - Server-side sorting, pagination control, and CSV export.
   - Modal forms for creating and editing assets with auto-complete datalists.
3. **Interactive AI Copilot (Chat Modal):**
   - WebSocket streaming connection with automatic reconnection.
   - In-browser voice dictation via Web Audio API and `MediaRecorder`.
   - Transcribed text editing before submission.
   - Multimodal document upload (PDF, JPG, PNG).
   - Browser-native Text-to-Speech (TTS) with automatic English and Indonesian language detection.
   - Visual model selector switch.

---

## 4. Database Schema & Data Models

The persistence layer runs on PostgreSQL. The primary database name is `poc_ai`.

```mermaid
erDiagram
    USERS ||--o{ CHAT : "creates"
    USERS ||--o{ TOOLS_HISTORY : "triggers"
    PERUSAHAAN ||--o{ USERS : "employs"
    PERUSAHAAN ||--o{ ASSETS : "owns"

    PERUSAHAAN {
        bigint id_perusahaan PK "IDENTITY"
        varchar nama_perusahaan
    }

    USERS {
        text id PK "UUID"
        text username "UNIQUE"
        text email "UNIQUE"
        text password_hash
        text role "admin | operator | viewer"
        text company
        bigint id_perusahaan FK
        timestamp created_at
        timestamp updated_at
    }

    ASSETS {
        varchar asset_id PK "AST-XXXXXXXX"
        varchar name
        varchar category
        varchar brand
        varchar model_type
        varchar purchase_date
        varchar purchase_price
        varchar location
        varchar created_at
        varchar perusahaan
        varchar status "active | inactive"
    }

    CHAT {
        varchar id PK "UUID"
        boolean user "true=User, false=System"
        text chat
        varchar created_at
        bigint user_id FK
        varchar username
        varchar role
        varchar models
    }

    TOOLS_HISTORY {
        text id PK "UUID"
        text perusahaan
        text tools
        text created_by_user_id
        text created_by_username
        text created_by_role
        timestamp created_at
    }
```

### Table Specifications

#### 1. `public.users`
Stores user credentials, access roles, and tenant associations.
* `id` (`text`, Primary Key): Generated UUID string.
* `username` (`text`, Not Null, Unique): Unique login name.
* `email` (`text`, Not Null, Unique): Validated email address.
* `password_hash` (`text`, Not Null): Bcrypt encrypted password.
* `role` (`text`, Default `'operator'`): Access tier (`admin`, `operator`, `viewer`).
* `company` (`text`): Associated organization name.
* `id_perusahaan` (`bigint`): Foreign identifier referencing `public.perusahaan`.
* `created_at` (`timestamp with time zone`): Account creation timestamp.
* `updated_at` (`timestamp with time zone`): Last update timestamp.

#### 2. `public.perusahaan`
Tenant directory containing recognized corporate entities.
* `id_perusahaan` (`bigint`, Primary Key, Generated Always As Identity).
* `nama_perusahaan` (`varchar(125)`): Legal or display name of the company.

#### 3. `public.assets`
Central IT inventory catalog.
* `asset_id` (`varchar(125)`, Primary Key): Canonical format `AST-XXXXXXXX`.
* `name` (`varchar(125)`, Not Null): Name of the asset (e.g., *MacBook Pro 16*).
* `category` (`varchar(125)`, Not Null): Category classification (e.g., *IT Equipment > Laptop*).
* `brand` (`varchar(125)`): Manufacturer (e.g., *Apple*, *Dell*, *Lenovo*).
* `model_type` (`varchar(125)`): Specific hardware model or configuration.
* `purchase_date` (`varchar(125)`): Purchase date string (`YYYY-MM-DD`).
* `purchase_price` (`varchar(125)`): Cost value formatted as numeric or string.
* `location` (`varchar(125)`): Physical location or warehouse assignment.
* `created_at` (`varchar(125)`): ISO 8601 or RFC 3339 creation timestamp.
* `perusahaan` (`varchar(125)`): Multi-tenant ownership identifier.
* `status` (`varchar(125)`, Default `'active'`): Operational status.

#### 4. `public.chat`
Historical message log for conversational audits and multi-turn context.
* `id` (`varchar(125)`, Primary Key): Message UUID.
* `user` (`boolean`): `true` if authored by a human user; `false` if generated by AI.
* `chat` (`text`): The textual payload of the message.
* `created_at` (`varchar(125)`): Formatted timestamp (`YYYY-MM-DD HH:MM:SS`).
* `user_id` (`bigint`): Foreign identifier of the user (nullable).
* `username` (`varchar(125)`): Sender display name or identifier.
* `role` (`varchar(125)`): Sender role (`admin`, `operator`, `system`).
* `models` (`varchar(125)`): AI model utilized for synthesis (e.g., `qwen`, `smolLM`).

#### 5. `public.tools_history`
Audit ledger tracking all tool executions triggered by AI agents.
* `id` (`text`, Primary Key): Execution UUID.
* `perusahaan` (`text`): Tenant organization context.
* `tools` (`text`): Name of the executed tool.
* `created_by_user_id` (`text`): User identifier who prompted the action.
* `created_by_username` (`text`): Username.
* `created_by_role` (`text`): User role at time of execution.
* `created_at` (`timestamp with time zone`): Execution timestamp.

---

## 5. API Reference & Interface Specifications

### 5.1 Authentication Endpoints

#### `POST /api/auth/register`
Creates a new user account and returns a 24-hour JWT token.
* **Access:** Public
* **Request Body:**
  ```json
  {
    "username": "johndoe",
    "email": "johndoe@company.com",
    "password": "securepassword123",
    "role": "operator",
    "company": "PT Qtera Mandiri",
    "id_perusahaan": 1
  }
  ```
* **Response (201 Created):**
  ```json
  {
    "token": "eyJhbGciOiJIUzI1NiIsIn...",
    "user": {
      "id": "c8942b0f-8c31-41be-8f78-7a5f663457a4",
      "username": "johndoe",
      "email": "johndoe@company.com",
      "role": "operator",
      "company": "PT Qtera Mandiri",
      "id_perusahaan": 1,
      "created_at": "2026-09-01T15:00:00Z"
    }
  }
  ```

#### `POST /api/auth/login`
Authenticates existing credentials.
* **Access:** Public
* **Request Body:**
  ```json
  {
    "username": "johndoe",
    "password": "securepassword123"
  }
  ```
* **Response (200 OK):** Identical to registration response.

#### `GET /api/auth/me`
Retrieves authenticated user profile from token.
* **Access:** Protected (`Authorization: Bearer <token>`)
* **Response (200 OK):** User profile object.

#### `GET /api/perusahaan`
Retrieves the list of registered companies.
* **Access:** Public
* **Response (200 OK):**
  ```json
  [
    { "id_perusahaan": 1, "nama_perusahaan": "PT Qtera Mandiri" },
    { "id_perusahaan": 2, "nama_perusahaan": "PT Teknologi Nusantara" }
  ]
  ```

---

### 5.2 Asset Management Endpoints

#### `GET /api/assets`
Lists, searches, and paginates assets scoped to the user's company.
* **Access:** Protected / Session-scoped
* **Query Parameters:**
  * `search` (optional): Substring search on name, asset ID, brand, or category.
  * `category` (optional): Comma-separated categories.
  * `brand` (optional): Comma-separated brands.
  * `location` (optional): Comma-separated locations.
  * `sort` (optional): Sort column (`name`, `category`, `brand`, `purchaseDate`, `createdAt`).
  * `order` (optional): `ASC` or `DESC` (default: `DESC`).
  * `page` (optional): Page number (default: `1`).
  * `pageSize` (optional): Items per page (default: `10`, or `'all'`).
* **Response (200 OK):**
  ```json
  {
    "assets": [
      {
        "assetId": "AST-A1B2C3D4",
        "name": "ThinkPad T14 Gen 4",
        "category": "IT Equipment > Laptop",
        "brand": "Lenovo",
        "modelType": "21AH00BCID",
        "purchaseDate": "2026-03-15",
        "purchasePrice": "19500000",
        "location": "HQ Office 4th Floor",
        "createdAt": "2026-03-15T10:00:00Z",
        "perusahaan": "PT Qtera Mandiri",
        "status": "active"
      }
    ],
    "pagination": {
      "page": 1,
      "pageSize": 10,
      "totalItems": 1,
      "totalPages": 1
    }
  }
  ```

#### `GET /api/assets/options`
Fetches distinct categories, locations, and brands present in the user's company records.
* **Access:** Protected / Session-scoped
* **Response (200 OK):**
  ```json
  {
    "categories": ["IT Equipment > Laptop", "Monitor", "Furniture"],
    "locations": ["HQ Office 4th Floor", "Warehouse A"],
    "brands": ["Apple", "Dell", "Lenovo"]
  }
  ```

#### `POST /api/assets`
Manually registers a new asset.
* **Access:** Protected (`admin`, `operator`)
* **Request Body:**
  ```json
  {
    "name": "Dell UltraSharp 27 Monitor",
    "category": "Monitor",
    "brand": "Dell",
    "modelType": "U2723QE",
    "purchaseDate": "2026-08-10",
    "purchasePrice": 8200000,
    "location": "Studio Room"
  }
  ```
* **Response (201 Created):** Returns generated `assetId`, `status`, and `createdAt`.

#### `PATCH /api/assets/:id`
Updates an existing asset record.
* **Access:** Protected (`admin`, `operator`)
* **Response (200 OK):** `{"ok": true, "assetId": "AST-A1B2C3D4"}`

#### `DELETE /api/assets/:id`
Deletes an asset record.
* **Access:** Protected (`admin` role required)
* **Response (204 No Content)**

---

### 5.3 Messaging, Chat & Audio Endpoints

#### `POST /message`
Submits a user message and triggers the AI orchestration workflow.
* **Access:** Protected (`Authorization: Bearer <token>`)
* **Content Types:** `application/json` or `multipart/form-data` (with file `attachment`).
* **Fields:**
  * `chat_id`: User or session identifier.
  * `text`: The query or instruction.
  * `models`: AI model selection (`qwen`, `smolLM`).
  * `attachment` (optional): Binary file (PDF, PNG, JPG).
* **Behavior:**
  1. Broadcasts the user message immediately across WebSocket `/ws/:id`.
  2. Persists the message in `public.chat`.
  3. Forwards request to `http://127.0.0.1:8000/rag`.
  4. Receives synthesized AI response.
  5. Broadcasts AI response across WebSocket with a brief debounce.
  6. Persists AI message in `public.chat`.

#### `POST /transcribe`
Transcribes audio without automatically submitting a message.
* **Access:** Protected
* **Payload:** Multipart form with field `audio` (WAV/WEBM file).
* **Response (200 OK):** `{"ok": true, "text": "Transcribed text content"}`

#### `POST /message/audio`
Transcribes uploaded audio and immediately triggers the chat pipeline.
* **Access:** Protected
* **Payload:** Multipart form with `audio`, `chat_id`, and `models`.

#### `GET /messages/:id`
Retrieves chat history for a specific username or session identifier.
* **Access:** Protected
* **Response (200 OK):** Array of historical message objects.

#### `GET /ws/:id`
WebSocket connection endpoint.
* **Access:** Authenticated via query parameter `?token=<jwt>` or header.
* **Protocol:** Text messages in JSON format.

---

### 5.4 AI Engine Endpoint (`eiai_go`)

#### `ALL /rag`
Processes natural language queries through MCP tools and returns a synthesized answer.
* **Host:** `http://127.0.0.1:8000/rag`
* **Parameters:** `query`, `models`, `role`, `token`, `chat_id`, `attachment`.
* **Output:** Plain text natural language response.

---

## 6. Security, Multi-Tenancy & Access Control

### 6.1 Role-Based Access Control (RBAC) Matrix

| Resource / Action | Admin | Operator | Viewer |
|:---|:---:|:---:|:---:|
| **View Asset Catalog (`GET /api/assets`)** | ✅ Allowed | ✅ Allowed | ✅ Allowed |
| **Download Asset CSV (`GET /api/assets/download`)** | ✅ Allowed | ✅ Allowed | ✅ Allowed |
| **Create Asset (`POST /api/assets`)** | ✅ Allowed | ✅ Allowed | ❌ Forbidden |
| **Update Asset (`PATCH /api/assets/:id`)** | ✅ Allowed | ✅ Allowed | ❌ Forbidden |
| **Delete Asset (`DELETE /api/assets/:id`)** | ✅ Allowed | ❌ Forbidden | ❌ Forbidden |
| **AI Tool: `get_assets`** | ✅ Permitted | ✅ Permitted | ✅ Permitted |
| **AI Tool: `add_asset`** | ✅ Permitted | ✅ Permitted | ❌ Denied |
| **AI Tool: `update_asset`** | ✅ Permitted | ✅ Permitted | ❌ Denied |
| **AI Tool: `delete_asset`** | ✅ Permitted | ❌ Denied | ❌ Denied |
| **Record Tool History (`POST /api/Tools`)** | ✅ Allowed | ✅ Allowed | ✅ Allowed |

### 6.2 Multi-Tenancy Data Segregation
* Every user is bound to a single company (`perusahaan`).
* When querying records, the backend automatically appends:
  ```sql
  WHERE (perusahaan ILIKE $1 OR perusahaan = '')
  ```
* During asset mutations (`UPDATE`, `DELETE`), the backend queries the target record first to ensure `existingCompany == userCompany`. If a mismatch is detected, the request is terminated with `HTTP 403 Forbidden`.
* In AI agent tool calls, credentials injected into the MCP server propagate the company identity, ensuring the LLM cannot access or mutate assets outside the requesting user's organization.

---

## 7. Environment Configuration & Deployment

### 7.1 Prerequisites

Ensure the following tools and runtimes are installed on the host system:
1. **Go Runtime:** Version 1.21 or higher.
2. **Node.js & npm:** Node.js v18+ and npm v9+.
3. **PostgreSQL Server:** Version 14+ running on port `5432`.
4. **FFmpeg:** Installed and available in the system `PATH` (used by audio conversion).
5. **Whisper-CLI:** Compiled binary placed in [server/GO/bin/](file:///c:/Users/user/OneDrive/Documents/proyek_DH/QTERA/pocAI2/server/GO/bin/) or available in `PATH`.
6. **Tesseract OCR (Optional for Image OCR):** Installed and available in `PATH`.

---

### 7.2 Configuration Files (`.env`)

#### 1. Core Backend: [server/GO/.env](file:///c:/Users/user/OneDrive/Documents/proyek_DH/QTERA/pocAI2/server/GO/.env)
```ini
DATABASE_URL=postgresql://postgres:deeha@localhost:5432/poc_ai?sslmode=disable
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=deeha
DB_NAME=poc_ai
DB_SSLMODE=disable
JWT_SECRET=qtera-super-secret-jwt-key-poc-2026
EIAI_URL=http://127.0.0.1:8000/rag
WHISPER_BIN=./bin/whisper-cli.exe
WHISPER_MODEL=./models/ggml-base.en.bin
```

#### 2. AI Orchestrator: [eiai_go/.env](file:///c:/Users/user/OneDrive/Documents/proyek_DH/QTERA/pocAI2/eiai_go/.env)
```ini
BASE_URL=http://127.0.0.1:3000/api
PORT=8000
BASE_AI_URL=https://openrouter.ai/api/v1
AI_KEY=sk-or-v1-YOUR_OPENROUTER_API_KEY
```

---

### 7.3 Step-by-Step Execution Guide

#### Step 1: Initialize Database
Restore the database schema from the root dump:
```bash
# Using psql or pg_restore
pg_restore -U postgres -d poc_ai -v database.sql
# Or initialize using psql
psql -U postgres -d poc_ai -f database.sql
```

#### Step 2: Start Core Backend (`server/GO`)
```bash
cd server/GO
go mod download
go run main.go
# Server listens on http://0.0.0.0:3000
```

#### Step 3: Start AI Orchestrator (`eiai_go`)
The AI orchestrator will automatically launch `mcp_server` over Stdio upon initialization:
```bash
cd eiai_go
go mod download
go run main.go
# Server listens on http://0.0.0.0:8000
```

#### Step 4: Start Frontend Client (`ai-registration-frontend`)
```bash
cd ai-registration-frontend
npm install
npm run dev
# Vite dev server runs at http://localhost:5173
```

---

## 8. Directory & Repository Structure

```
pocAI2/
├── database.sql                     # PostgreSQL database backup and schema dump
├── TECHNICAL_DOCUMENTATION.md       # Primary technical architecture specification
│
├── server/                          # Core REST & WebSocket Backend
│   └── GO/
│       ├── .env                     # Backend configuration (DB credentials, Whisper paths)
│       ├── main.go                  # Fiber app entry point & routing configuration
│       ├── controller/              # HTTP handlers & business logic
│       │   ├── auth.go              # User registration, login & JWT generation
│       │   ├── assets.go            # Assets CRUD, sorting, filtering & options
│       │   ├── SendMessage.go       # WebSocket hub, chat persistence & RAG proxy
│       │   ├── getMessage.go        # Historical chat retrieval
│       │   ├── transcribe.go        # FFmpeg audio conversion & whisper-cli STT
│       │   ├── perusahaan.go        # Tenant company directory handlers
│       │   ├── tools.go             # Tools execution audit logger
│       │   └── struck.go            # Central pgxpool database connection
│       ├── middleware/              # Fiber middleware (JWT auth, WebSocket upgrades)
│       ├── models/                  # Go structs mapping to database entities
│       │   ├── assets.go            # Asset & session models
│       │   ├── chat.go              # Chat entity model
│       │   ├── user.go              # User profile & credentials model
│       │   └── ggml-base.en.bin     # Quantized Whisper English speech model
│       └── bin/                     # Platform binaries (whisper-cli.exe)
│
├── eiai_go/                         # AI & RAG Orchestration Service
│   ├── .env                         # AI API keys & endpoint definitions
│   ├── main.go                      # Fiber HTTP server exposing /rag
│   └── controller/                  # AI reasoning & execution pipeline
│       ├── models.go                # OpenRouter LLM interface, reasoning loops & prompts
│       ├── toolCall.go              # Tool execution, Caveman token compression & RBAC
│       ├── identifier.go            # Smart identifier matching & resolver engine
│       ├── extract.go               # PDF text parser & Tesseract OCR engine
│       └── decode_jwt.go            # User context recovery & chat history extractor
│
├── mcp_server/                      # Model Context Protocol (MCP) Server
│   ├── go.mod                       # MCP server dependencies
│   └── server.go                    # Stdio MCP tool definitions & REST proxies
│
└── ai-registration-frontend/        # Client Application (React 18 + Vite)
    ├── index.html                   # HTML entry point with Google Fonts & Phosphor Icons
    ├── vite.config.js               # Dev server configuration with backend proxy rules
    ├── package.json                 # Frontend dependencies
    └── src/
        ├── main.jsx                 # React root component initialization
        ├── App.jsx                  # Main dashboard, asset tables, filters & auth views
        ├── Chat.jsx                 # AI chat dialog with voice recording & TTS
        ├── AuthContext.jsx          # React authentication state context
        ├── audioRecorder.js         # Web Audio API audio recording utility
        ├── services.js              # API client wrapper for backend communications
        └── App.css                  # Custom styling & modern UI design system
```

---

## 9. Conclusion

The **QTERA Asset Management & AI Platform** demonstrates a production-grade pattern for integrating localized AI tool-calling with existing transactional enterprise software. By enforcing multi-tenancy at both the HTTP layer and within the MCP tool interface, the system ensures data security while providing flexible conversational and multimodal interfaces for managing enterprise assets.
