from fastapi import FastAPI

app = FastAPI()
# runs on 8000 by default
# http://localhost:8000

@app.get("/api/")
def read_root():
    return {"Hello": "World"}

@app.get("/api/health")
def health_check():
    return {"status": "healthy"}

@app.get("/items/{item_id}")
def read_item(item_id: int, q: str | None = None):
    return {"item_id": item_id, "q": q}