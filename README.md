# AWS Local Dashboard

<img width="1500" height="812" alt="Screenshot 2026-01-09 at 6 52 35 PM" src="https://github.com/user-attachments/assets/9f563f8a-7de9-4459-b651-1a5a60036d40" />



> A production-ready, local-only AWS cost and resource visibility dashboard. No AWS Console login required.

[![Docker](https://img.shields.io/badge/Docker-Ready-blue?logo=docker)](https://docker.com)
[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://golang.org)
[![React](https://img.shields.io/badge/React-18+-61DAFB?logo=react)](https://reactjs.org)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)

---

## ✨ Features

| Feature | Description |
|---------|-------------|
| 💰 **Cost Explorer** | View total spend, credits applied, net cost with date range filters |
| 💱 **Currency Converter** | 30+ currencies with searchable dropdown and editable exchange rates |
| 🖥️ **Resource Browser** | Browse EC2, VPC, EIP, S3, RDS, Rekognition across all regions |
| ⌨️ **CLI Runner** | Execute read-only AWS commands with safety checks |
| 👤 **Multi-Profile** | Switch AWS profiles or add custom credentials via UI |
| 🔄 **Smart Caching** | 60-second TTL cache with manual refresh option |
| 🌙 **Dark Theme** | Professional dark UI inspired by AWS Console |

---

## 🚀 Quick Start

### Option 1: Docker Hub (Easiest)

```bash
docker run -d \
  -p 9090:8080 \
  -v ~/.aws:/root/.aws:ro \
  manish2538/aws-local-dashboard
```

Then open **http://localhost:9090**

### Option 2: Build & Run Script

```bash
./run.sh
```

This automatically:
- ✅ run the makefile command


### Option 3: Using Make

```bash
make dev
```

### Option 4: Manual Docker Build

```bash
docker build -t aws-local-dashboard .

docker run -d \
  --name aws-dashboard \
  -p 8080:8080 \
  -v ~/.aws:/root/.aws:ro \
  -v $(pwd)/data:/app/data \
  aws-local-dashboard
```

Then open **http://localhost:8080**

---

## 📸 What You Get

### Cost Explorer
- **Total Spend** – Current month or custom date range
- **Credits Applied** – Free tier and promotional credits
- **Net Cost** – After credits
- **Service Breakdown** – Clickable chart and table
- **Cost Filters** – Min/max cost range filtering

### Currency Converter
- **30+ Currencies** – USD, EUR, GBP, INR, JPY, CNY, and more
- **Searchable** – Type currency code or name to find
- **Editable Rates** – Click on rate to enter custom value
- **Persistent** – Saved to browser localStorage

### Resource Browser
| Service | What You See |
|---------|-------------|
| EC2 | Instance ID, Name, State, Type, AZ, IPs |
| VPC | VPC ID, CIDR, State, Default flag |
| EIP | Allocation ID, Public IP, Associations |
| S3 | Bucket Name, Creation Date |
| RDS | DB Identifier, Engine, Status, Endpoint |
| Rekognition | Collection ID, Face Model Version |

- **All Regions** – Parallel fetch across all AWS regions
- **Filters** – EC2 state filter (running/stopped/etc.)

### CLI Runner
- **Predefined Commands** – Curated list of safe read-only commands
- **Raw Command Input** – Enter any describe/list/get command
- **Safety Checks** – Blocks create/delete/terminate operations
- **Output Display** – Shows exact command executed + JSON response

### Profile Management
- **System Credentials** – Uses `~/.aws` automatically
- **Custom Profiles** – Add Access Key ID + Secret via UI
- **Profile Switching** – Dropdown to switch active profile
- **Persistent Storage** – Profiles saved to local file

---

## 🛠️ Local Development

### Prerequisites

- Go 1.22+
- Node.js 20+
- AWS CLI installed and configured

### Development Mode

```bash
# Run both backend and frontend
make dev

# Or separately:
make backend   # Go server on :8080
make frontend  # Vite dev server on :5173 (proxies to backend)
```

### Production Build (without Docker)

```bash
make run-backend-with-build
# Serves on http://localhost:8080
```

---

## 📁 Project Structure

```
aws-web/
├── backend/
│   ├── cmd/server/main.go          # Entry point
│   ├── internal/
│   │   ├── httpserver/server.go    # HTTP routes & handlers
│   │   ├── awscli/
│   │   │   ├── executor.go         # AWS CLI wrapper
│   │   │   ├── cost_service.go     # Cost Explorer queries
│   │   │   └── resource_service.go # Resource describe calls
│   │   ├── services/services.go    # Service interfaces
│   │   ├── types/types.go          # Shared DTOs
│   │   ├── cache/cache.go          # In-memory TTL cache
│   │   ├── profiles/manager.go     # Profile management
│   │   └── commands/config.go      # CLI command runner
│   └── command-config.json         # Predefined safe commands
│
├── frontend/
│   ├── src/
│   │   ├── pages/
│   │   │   ├── DashboardPage.tsx       # Cost Explorer
│   │   │   ├── ServiceDetailPage.tsx   # Resource drilldown
│   │   │   ├── ResourcesOverviewPage.tsx
│   │   │   └── CommandRunnerPage.tsx
│   │   ├── components/
│   │   │   ├── ProfileBar.tsx          # Profile switcher
│   │   │   └── CurrencySelector.tsx    # Currency converter
│   │   ├── context/CurrencyContext.tsx
│   │   ├── utils/currency.ts           # Exchange rates
│   │   ├── api/client.ts               # API client
│   │   └── styles.css                  # Global styles
│   └── index.html
│
├── Dockerfile              # Multi-stage build
├── Makefile                # Build commands
├── run.sh                  # Docker build & run script
└── README.md
```


## ⚙️ Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `STATIC_DIR` | `./static` | Frontend static files directory |
| `CACHE_TTL_SECONDS` | `60` | Cache time-to-live in seconds |
| `COMMAND_CONFIG_PATH` | `./command-config.json` | Path to predefined commands |
| `PROFILE_STORE_PATH` | `./.aws-local-dashboard-profiles.json` | Profile storage file |
| `AWS_PROFILE` | *(none)* | AWS CLI profile to use |

### Custom Port

```bash
PORT=3000 ./run.sh
```

---

## 🔐 Required IAM Permissions

Create an IAM policy with these permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "AWSLocalDashboardReadOnly",
      "Effect": "Allow",
      "Action": [
        "ce:GetCostAndUsage",
        "ec2:DescribeInstances",
        "ec2:DescribeVpcs",
        "ec2:DescribeAddresses",
        "ec2:DescribeRegions",
        "ec2:DescribeVolumes",
        "s3:ListAllMyBuckets",
        "rds:DescribeDBInstances",
        "rekognition:ListCollections",
        "iam:ListUsers",
        "iam:ListRoles",
        "cloudwatch:DescribeAlarms",
        "sts:GetCallerIdentity"
      ],
      "Resource": "*"
    }
  ]
}
```

> **Note:** Cost Explorer must be enabled in your AWS account. Enable it at:  
> AWS Console → Billing → Cost Explorer → Enable Cost Explorer

---

## 🐛 Troubleshooting

### Container won't start

```bash
docker logs aws-dashboard
```

### AWS credentials not found

1. Check if `~/.aws` exists:
   ```bash
   ls -la ~/.aws
   ```

2. Verify AWS CLI works:
   ```bash
   aws sts get-caller-identity
   ```

3. Or add credentials via the dashboard UI (click "Add Profile")

### Cost Explorer shows error

- Cost Explorer must be **enabled** in AWS Console
- Takes up to **24 hours** to activate after enabling
- Verify you have `ce:GetCostAndUsage` permission

### Port already in use

```bash
# Use different port
PORT=3000 ./run.sh

# Or stop existing container
make docker-stop
```

### Slow "All Regions" queries

This is expected – the dashboard queries up to 20+ regions in parallel. Results are cached for 60 seconds.

---

## 📝 Make Commands

| Command | Description |
|---------|-------------|
| `make dev` | Run backend + frontend for development |
| `make backend` | Run Go backend only |
| `make frontend` | Run Vite dev server only |
| `make docker` | Build and run Docker container |
| `make docker-build` | Build Docker image only |
| `make docker-run` | Run container (image must exist) |
| `make docker-stop` | Stop and remove container |
| `make docker-logs` | View container logs |
| `make clean` | Remove all build artifacts |

---

## 📦 Publishing to Docker Hub

To publish this image to Docker Hub for easy distribution:

### Step 1: Login to Docker Hub

```bash
docker login
```

### Step 2: Use the publish script

```bash
# Set your Docker Hub username
DOCKER_USERNAME=yourusername ./publish.sh

# Or with a specific version
DOCKER_USERNAME=yourusername VERSION=1.0.0 ./publish.sh
```

### Step 3: Share with users

After publishing, users can run with a single command:

```bash
docker run -d -p 8080:8080 -v ~/.aws:/root/.aws:ro yourusername/aws-local-dashboard
```

---

## 🏗️ Tech Stack

| Layer | Technology |
|-------|------------|
| **Backend** | Go 1.22, net/http, os/exec |
| **Frontend** | React 18, TypeScript, Vite, Recharts |
| **Styling** | Custom CSS (dark theme) |
| **Container** | Alpine Linux, AWS CLI |
| **Build** | Multi-stage Docker |

---

## 📄 License

MIT License - feel free to use, modify, and distribute.

---

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run `make dev` to test locally
5. Submit a pull request

---

⚠️ This tool uses your local AWS credentials via AWS CLI.
It does not store or transmit credentials.
