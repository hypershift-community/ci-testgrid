# CI TestGrid

A web application for visualizing and analyzing CI test results from OpenShift CI jobs. This project consists of two main components: a scraper that collects test data and a web UI that displays the results.

## Project Structure

```
.
├── scraper/     # Backend service for scraping CI test results
│   ├── db/      # Database operations
│   ├── processor/ # Test result processing
│   ├── scraper/   # Web scraping logic
│   └── types/     # Data type definitions
└── ui/          # Frontend web application
    ├── templates/ # HTML templates
    └── testgrid/  # UI handlers and logic
```

## Features

- Scrapes test results from OpenShift CI jobs
- Processes and stores test data in MongoDB
- Web interface for viewing test results
- Kubernetes deployment support

## Prerequisites

- Go 1.21 or later
- Docker
- Kubernetes cluster (for deployment)
- MongoDB instance

## Getting Started

### Building the Scraper

```bash
cd scraper
make build
```

### Building the UI

```bash
cd ui
make build
```

### Running Locally

1. Start MongoDB:
```bash
docker run -d -p 27017:27017 mongo:latest
```

2. Run the scraper:
```bash
cd scraper
./bin/scraper
```

3. Run the UI:
```bash
cd ui
./bin/ui
```

The UI will be available at `http://localhost:8080`

### Kubernetes Deployment

Both components can be deployed to Kubernetes using the provided manifests in their respective `k8s/` directories.

## Development

### Project Structure

- `scraper/`: Contains the backend service that scrapes CI test results
  - `main.go`: Entry point for the scraper service
  - `db/`: MongoDB operations and connection handling
  - `processor/`: Test result processing logic
  - `scraper/`: Web scraping implementation
  - `types/`: Data structures and type definitions

- `ui/`: Contains the web interface
  - `main.go`: Entry point for the web server
  - `templates/`: HTML templates for the web interface
  - `testgrid/`: UI handlers and business logic

### Adding New Features

1. Create a new branch for your feature
2. Make your changes
3. Update tests if necessary
4. Submit a pull request

## License

This project is licensed under the Apache License 2.0 - see the LICENSE file for details. 