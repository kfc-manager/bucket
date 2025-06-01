FROM python:3.13.3

COPY ./requirements.txt ./requirements.txt
RUN pip install -r requirements.txt

COPY ./ ./
RUN  rm requirements.txt

CMD ["pytest", "test_bucket.py"]
