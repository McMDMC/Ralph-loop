# 🤖 Ralph Orchestrator

[![Cloud Build Status](https://github.com/mmcmorris47/Ralph-loop/actions/workflows/main.yml/badge.svg)](https://github.com/mmcmorris47/Ralph-loop/actions)
![Go Version](https://img.shields.io/badge/Go-1.21-00ADD8?style=flat&logo=go)
![Platform](https://img.shields.io/badge/Google_Cloud-Cloud_Run-4285F4?style=flat&logo=google-cloud)

A high-availability Go-based orchestrator deployed as a serverless microservice on Google Cloud.

## 🚀 Overview
Ralph Orchestrator is a portfolio piece demonstrating modern cloud-native development practices. It features a fully automated CI/CD pipeline and serves a dynamic landing page using embedded assets.

- **Frontend:** HTML5/CSS3 (Embedded in Go binary)
- **Backend:** Go (Golang) 1.21
- **Cloud Infrastructure:** Google Cloud Platform (GCP)
- **CI/CD:** Google Cloud Build & GitHub Actions

## 🛠️ Tech Stack
- **Language:** Go (Net/HTTP, Embed)
- **Compute:** [Google Cloud Run](https://cloud.google.com/run) (Serverless)
- **Registry:** [Artifact Registry](https://cloud.google.com/artifact-registry)
- **Pipeline:** [Cloud Build Triggers](https://cloud.google.com/build/docs/automating-builds/create-manage-triggers)

## 📦 Local Development
To run this project locally, ensure you have Go 1.21+ installed.

1. Clone the repository:
   ```bash
   git clone [https://github.com/mmcmorris47/Ralph-loop.git](https://github.com/mmcmorris47/Ralph-loop.git)