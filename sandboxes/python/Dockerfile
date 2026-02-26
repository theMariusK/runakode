# python-runner
FROM python:3.12-alpine

RUN adduser -D -u 1000 sandbox

WORKDIR /sandbox

USER sandbox

ENTRYPOINT ["/bin/sh", "-c"]
CMD ["python3 main.py"]
