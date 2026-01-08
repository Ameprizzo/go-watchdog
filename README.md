# 🐕 Go-Watchdog

[![Go Report Card](https://goreportcard.com/badge/github.com/Ameprizzo/go-watchdog)](https://goreportcard.com/report/github.com/Ameprizzo/go-watchdog)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
![Go Version](https://img.shields.io/github/go-mod/go-version/Ameprizzo/go-watchdog)
![Docker Image](https://img.shields.io/badge/docker-ready-blue?logo=docker)

**Go-Watchdog** is a high-performance, concurrent service monitoring tool built with Go. It allows you to monitor the uptime of your websites and APIs in real-time through a sleek web dashboard and automated status checks.



## 🚀 Features

* **Concurrent Monitoring:** Leverages Go's **Goroutines** to perform health checks in parallel, ensuring high performance regardless of the number of targets.
* **Modern Web UI:** A clean, responsive dashboard to visualize the health and latency of your services.
* **Dockerized Deployment:** Easily portable and ready to deploy as a lightweight container.
* **Real-time Alerts:** Instant console logging for downtime detection (extendable to Discord/Slack).
* **JSON Configured:** Simple management of target URLs via a central configuration file.

## 🛠 Tech Stack

* **Language:** Go (Golang)
* **Frontend:** HTML5, CSS (Tailwind/Standard), JavaScript
* **Infrastructure:** Docker
* **Concurreny:** Channels & WaitGroups

## 📂 Project Structure

```text
go-watchdog/
├── cmd/
│   └── watchdog/
│       └── main.go       # Entry point & HTTP Server setup
├── internal/
│   ├── monitor/
│   │   └── monitor.go    # Concurrency logic & HTTP pinger
│   └── notifier/
│       └── notifier.go   # Alerting & Logging logic
├── web/
│   ├── static/           # CSS & JS files
│   └── templates/        # HTML templates for the UI
├── config.json           # List of URLs to monitor
├── Dockerfile            # Container configuration
└── go.mod                # Dependency management
```
## 🚦 Getting Started

### Prerequisites
* **Go 1.21+**
* **Docker** (Optional)

### Installation & Local Run

1.  **Clone the repository:**
    ```bash
    git clone [https://github.com/Ameprizzo/go-watchdog.git](https://github.com/Ameprizzo/go-watchdog.git)
    cd go-watchdog
    ```

2.  **Initialize the module (if not already done):**
    ```bash
    go mod tidy
    ```

3.  **Run the application:**
    ```bash
    go run cmd/watchdog/main.go
    ```

### Running with Docker

```bash
# Build the image
docker build -t go-watchdog .

# Run the container
docker run -p 8080:8080 go-watchdog
```
## 📈 Roadmap
- [ ] **Historical Tracking:** Implement uptime percentage tracking with a lightweight SQLite database.
- [ ] **Alerting:** Add support for Discord and Slack webhooks for instant notifications.
- [ ] **Custom Intervals:** Allow per-service check intervals (e.g., check API every 10s, but Blog every 5m).
- [ ] **Unit Testing:** Add comprehensive tests for the monitoring and notification logic.

## 👤 Author
**Amedeus Primi Lyakurwa**
* **GitHub:** [@Ameprizzo](https://github.com/Ameprizzo)

---
*Developed as a portfolio project to demonstrate Go concurrency, clean architecture, and system monitoring.*