FROM golang:latest as build
RUN mkdir -p /usr/local/ethlogspy
COPY *.go /usr/local/ethlogspy/
COPY configs/config.yml /usr/local/ethlogspy/
COPY go.mod /usr/local/ethlogspy/
COPY go.sum /usr/local/ethlogspy/
COPY scripts/entrypoint.sh /usr/local/ethlogspy/
WORKDIR /usr/local/ethlogspy
RUN go mod tidy && go build -o ethlogspy *.go

FROM golang:latest 
RUN mkdir -p /usr/local/ethlogspy
WORKDIR /usr/local/ethlogspy
COPY --from=build /usr/local/ethlogspy/ethlogspy .
COPY --from=build /usr/local/ethlogspy/config.yml .
COPY --from=build /usr/local/ethlogspy/entrypoint.sh .
RUN chmod +x entrypoint.sh
EXPOSE 8080
ENTRYPOINT [ "./entrypoint.sh" ]