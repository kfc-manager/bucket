# Bucket :bucket:

This project is an implementation of basic operations of the AWS S3 API. It implements:

- `create_bucket`
- `get_object`
- `put_object`
- `delete_object`

Because it mirrors the AWS API it is compatible with SDKs such as `boto3`. This makes it suitable for local development, test mocking, or
lightweight self-hosted object storage.

## Note :speech_balloon:

This implementation does not provide any built-in mechanisms for data redundancy. It assumes that data durability is handled by the storage
layer of the underlying host system, such as `RAID`.

## Getting Started :rocket:

```bash
docker build -t bucket .
docker run -p 8000:8000 -e ACCESS_KEY="<your_access_key>" -e SECRET_KEY="<your_secret_key>" -d bucket
```

You can then interact with the bucket using the official AWS SDK:

```python
# pip install boto3==1.38.25
import boto3
from botocore.client import Config

s3 = boto3.client(
    "s3",
    endpoint_url=f"http://localhost:8000",
    aws_access_key_id="<your_access_key>",
    aws_secret_access_key="<your_secret_key>",
    config=Config(signature_version="s3v4"),
    region_name="us-east-1",
)

bucket_name = "test-bucket"
object_key = "test.txt"
body = b"hello world!"
s3.create_bucket(Bucket=bucket_name)
s3.put_object(Bucket=bucket_name, Key=object_key, Body=body)
response = s3.get_object(Bucket=bucket_name, Key=object_key)
print(body == response["Body"].read()) # True
```

## Motivation :bulb:

While solutions like `MinIO` offer similar S3-compatible storage APIs, they require a paid subscription for production use
(as of: June 1. 2025). This limitation was one of the main driving factors behind this project. The goal is also to provide a more
lightweight solution that supports only essential S3 operations.
