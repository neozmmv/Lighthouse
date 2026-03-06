# ---- build stage ----
FROM python:3.12-slim AS builder

WORKDIR /app

# installs dependencies in a separate stage to keep the final image smaller
COPY requirements.txt .
RUN pip install --no-cache-dir --prefix=/install -r requirements.txt


# ---- runtime stage ----
FROM python:3.12-slim

WORKDIR /app

# copy installed dependencies from builder stage
COPY --from=builder /install /usr/local

# copy application code
COPY . .

EXPOSE 8000

CMD ["fastapi", "run", "main.py", "--host", "0.0.0.0", "--port", "8000"]