FROM python:3

WORKDIR /usr/src/app
COPY ./app/telegram.py ./
COPY ./config/* ./config/

RUN pip install --no-cache-dir python-dotenv telethon

# Unbuffered stdout so logs appear immediately
ENV PYTHONUNBUFFERED=1

WORKDIR /usr/src/app/config
CMD [ "python", "../telegram.py" ]