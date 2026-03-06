import os
import boto3
from botocore.config import Config
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
from uuid import uuid4

app = FastAPI()

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)

_creds = dict(
    aws_access_key_id=os.getenv("MINIO_ROOT_USER", "lighthouse"),
    aws_secret_access_key=os.getenv("MINIO_ROOT_PASSWORD", "lighthouse_secret"),
    region_name="us-east-1",
    config=Config(signature_version="s3v4"),
)

# internal client — used for all S3 operations
s3 = boto3.client("s3", endpoint_url=os.getenv("MINIO_ENDPOINT", "http://minio:9000"), **_creds)

# public client — used only for presigned URL generation; must use the same host clients will PUT to
s3_public = boto3.client("s3", endpoint_url=os.getenv("MINIO_PUBLIC_ENDPOINT", "http://localhost:9000"), **_creds)

BUCKET = "lighthouse"


# starts multipart upload, returns the upload_id and presigned URLs for each chunk

class InitUploadRequest(BaseModel):
    filename: str
    total_chunks: int  # frontend needs to know how many chunks it will upload, so it can request the correct number of presigned URLs

# frontend sends the ETag from each upload response, along with the part number, so we can complete the multipart upload

class Part(BaseModel):
    part_number: int
    etag: str

class FinishUploadRequest(BaseModel):
    file_id: str
    upload_id: str
    parts: list[Part]  # frontend sends the ETag from each upload response, along with the part number

# aborts an upload if something went wrong

class AbortUploadRequest(BaseModel):
    file_id: str
    upload_id: str

@app.get("/api/health")
def health():
    return {"status": "ok"}

@app.post("/api/upload/init")
def upload_init(body: InitUploadRequest):
    file_id = str(uuid4())

    # starts multipart upload, returns the upload_id
    response = s3.create_multipart_upload(
        Bucket=BUCKET,
        Key=file_id,
        Metadata={"filename": body.filename},
    )
    upload_id = response["UploadId"]

    # presigned url for each chunk
    presigned_urls = []
    for part_number in range(1, body.total_chunks + 1):
        url = s3_public.generate_presigned_url(
            "upload_part",
            Params={
                "Bucket": BUCKET,
                "Key": file_id,
                "UploadId": upload_id,
                "PartNumber": part_number,
            },
            ExpiresIn=7200,  # 2 hours, can be adjusted based on expected upload times and chunk sizes
        )
        presigned_urls.append({"part_number": part_number, "url": url})

    return {
        "file_id": file_id,
        "upload_id": upload_id,
        "urls": presigned_urls,
    }

@app.post("/api/upload/finish")
def upload_finish(body: FinishUploadRequest):
    s3.complete_multipart_upload(
        Bucket=BUCKET,
        Key=body.file_id,
        UploadId=body.upload_id,
        MultipartUpload={
            "Parts": [
                {"PartNumber": p.part_number, "ETag": p.etag}
                for p in sorted(body.parts, key=lambda x: x.part_number)
            ]
        },
    )
    return {"file_id": body.file_id, "status": "ok"}

@app.post("/api/upload/abort")
def upload_abort(body: AbortUploadRequest):
    s3.abort_multipart_upload(
        Bucket=BUCKET,
        Key=body.file_id,
        UploadId=body.upload_id,
    )
    return {"status": "aborted"}