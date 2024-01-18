FROM golang:alpine as build
WORKDIR /src

# Copy Go code
COPY ./go.mod ./go.sum ./
COPY *.go ./

# Copy help
COPY ./help.md ./

RUN go mod download
RUN  go build -o /bin/modmail

FROM alpine:latest
COPY --from=build /bin/modmail /bin/modmail

EXPOSE 8080
CMD ["/bin/modmail"]