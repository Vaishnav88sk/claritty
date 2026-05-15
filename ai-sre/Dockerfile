FROM python:3.12-slim

# Set working directory
WORKDIR /app

# Install git and other system dependencies needed for compiling
RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# Copy only the requirements file first to leverage Docker cache
COPY requirements.txt .

# Install dependencies without cache
RUN pip install --no-cache-dir --upgrade pip && \
    pip install --no-cache-dir -r requirements.txt

# Copy the rest of the application
COPY . .

# Install the claritty_sre package globally inside the container
RUN pip install --no-cache-dir .

# Set environment variables for the container
ENV DB_PATH=/root/.claritty/claritty_sre.db
ENV RUNBOOKS_DIR=/app/claritty_sre/runbooks

# Entrypoint for the CLI
ENTRYPOINT ["clarctl"]
