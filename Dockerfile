FROM golang:latest as build
RUN mkdir -p /usr/local/ethlogspy
COPY *.go /usr/local/ethlogspy
COPY configs /usr/local/ethlogspy
COPY go.mod /usr/local/ethlogspy
COPY go.sum /usr/local/ethlogspy
WORKDIR /usr/local/ethlogspy
RUN mv scripts/entrypoint.sh .
RUN go mod tidy && go build -o ethlogspy *.go

FROM alpine:latest 
RUN mkdir -p /usr/local/ethlogspy
WORKDIR /usr/local/ethlogspy
COPY --from=build /usr/local/ethlogspy/ethlogspy .
COPY --from=build /usr/local/ethlogspy/configs .
COPY --from=build /usr/local/ethlogspy/entrypoint.sh .
RUN chmod +x entrypoint.sh
EXPOSE 8080
ENTRYPOINT [ "./entrypoint.sh" ]