FROM golang:latest

RUN mkdir -p /usr/local/ethlogspy
COPY . /usr/local/ethlogspy
WORKDIR /usr/local/ethlogspy
RUN mv scripts/entrypoint.sh .
RUN chmod +x entrypoint.sh
RUN go build -o ethlogspy *.go

EXPOSE 8080
ENTRYPOINT [ "./entrypoint.sh" ]
