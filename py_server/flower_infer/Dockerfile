FROM python:3.12-slim

WORKDIR /app

COPY py_server/flower_infer/requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt -i https://mirrors.cloud.tencent.com/pypi/simple

COPY py_server/flower_infer/flower_infer.py ./

EXPOSE 8082

CMD ["uvicorn", "flower_infer:app", "--host", "0.0.0.0", "--port", "8082"] 