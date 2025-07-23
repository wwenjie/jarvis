FROM python:3.12

WORKDIR /app

COPY py_server/requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

COPY py_server/flower_infer.py ./

EXPOSE 8082

CMD ["uvicorn", "flower_infer:app", "--host", "0.0.0.0", "--port", "8082"] 