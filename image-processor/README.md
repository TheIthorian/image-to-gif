# image-processor

Application to process images into a gif

## Install

This app uses docker. To run the application:

```sh
docker compose up --build
```

## Start

```sh
python3 -m flask run --host=0.0.0.0 --port=8080
```

## Api Keys

Set the `APP_KEY` (generated and set by you) in `.docker.env`

## Endpoints

| Method | Url                   | Description                        |
| ------ | --------------------- | ---------------------------------- |
| `GET`  | `api/health`          | Health check                       |
| `GET`  | `/`                   | Upload page                        |
| `POST` | `/upload`             | Takes the images and saves the gif |
| `POST` | `/uploads/output.gif` | The generated gif                  |
