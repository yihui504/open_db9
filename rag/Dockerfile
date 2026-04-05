FROM python:3.10-slim

WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    gcc \
    postgresql-client \
    && rm -rf /var/lib/apt/lists/*

# Copy project files from root context
COPY rag/pyproject.toml .
COPY rag/src/ ./src/
COPY rag/scripts/ ./scripts/
COPY rag/README.md .
COPY migrations/ ./migrations/

# Install Python dependencies
RUN pip install --no-cache-dir -e .

# Expose port
EXPOSE 8001

# Run the server
CMD ["uvicorn", "src.api.server:app", "--host", "0.0.0.0", "--port", "8001"]
