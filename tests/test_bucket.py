import os
import boto3
from botocore.client import Config


s3 = boto3.client(
    "s3",
    endpoint_url=f"http://{os.getenv('HOST')}",
    aws_access_key_id=os.getenv("ACCESS_KEY"),
    aws_secret_access_key=os.getenv("SECRET_KEY"),
    config=Config(signature_version="s3v4"),
    region_name="us-east-1",
)


def test_create_bucket():
    s3.create_bucket(Bucket="test-create-bucket")


def test_object_storage():
    bucket_name = "test-object-storage"
    object_key = "test.txt"
    body = b"hello world!"

    s3.create_bucket(Bucket=bucket_name)
    s3.put_object(Bucket=bucket_name, Key=object_key, Body=body)
    response = s3.get_object(Bucket=bucket_name, Key=object_key)

    assert body == response["Body"].read()


def test_delete_object():
    bucket_name = "test-delete-object"
    object_key = "test.txt"

    s3.create_bucket(Bucket=bucket_name)
    s3.put_object(Bucket=bucket_name, Key=object_key, Body=b"hello world!")
    s3.delete_object(Bucket=bucket_name, Key=object_key)
    try:
        s3.get_object(Bucket=bucket_name, Key=object_key)
        assert False, "Expected an exception when accessing a deleted object"
    except Exception:
        pass
