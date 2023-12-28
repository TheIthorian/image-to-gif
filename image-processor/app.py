from flask import (
    Flask,
    render_template,
    request,
    redirect,
    url_for,
    send_from_directory,
)
import os
import imageio
from PIL import Image
import numpy as np
import math

app = Flask(__name__)

UPLOAD_FOLDER = "uploads"
app.config["UPLOAD_FOLDER"] = UPLOAD_FOLDER


# Ensure the upload folder exists
if not os.path.exists(UPLOAD_FOLDER):
    os.makedirs(UPLOAD_FOLDER)


@app.route("/api/health")
def hello_world():
    return "ok"


@app.route("/")
def index():
    return render_template("./index.html")


@app.route("/uploads/<filename>")
def uploaded_file(filename):
    return send_from_directory(app.config["UPLOAD_FOLDER"], filename)


@app.route("/upload", methods=["POST"])
def upload_images():
    # Check if the POST request has the file part
    if "files" not in request.files:
        return redirect(request.url)

    delete_old_images(app.config["UPLOAD_FOLDER"])

    files = request.files.getlist("files")
    fps = request.form["fps"]
    scale_factor = int(request.form["scale-factor"] or "100") / 100

    uploaded_file_paths = []

    for file in files:
        if file.filename == "":
            return redirect(request.url)

        # Save the file to the server
        filename = os.path.join(app.config["UPLOAD_FOLDER"], file.filename)
        file.save(filename)
        uploaded_file_paths.append(filename)

    # Create a GIF from the uploaded images
    gif_path = create_gif(uploaded_file_paths, fps, scale_factor)

    print(gif_path)

    return render_template("./result.html", gif_path="output.gif")


# size = 128, 128


def create_gif(image_paths, fps=2, scale_factor=1):
    print("rescaling images")
    for image_path in image_paths:
        resize_image(image_path, scale_factor)

    print("generating gif")
    images = [imageio.imread(path) for path in image_paths]

    gif_path = os.path.join(app.config["UPLOAD_FOLDER"], "output.gif")
    imageio.mimsave(gif_path, images, fps=int(fps), format="GIF", loop=0)

    return gif_path


def resize_image(image_path, scale_factor):
    im = Image.open(image_path)
    size = (
        math.floor(im.width * float(scale_factor)),
        math.floor(im.height * float(scale_factor)),
    )
    print(size)
    im.thumbnail(size, Image.Resampling.LANCZOS)
    im.save(image_path)


def delete_old_images(directory):
    print(f"delete_old_images for {directory}")

    # Walk through the directory and its subdirectories
    for foldername, _, filenames in os.walk(directory):
        for filename in filenames:
            file_path = os.path.join(foldername, filename)

            try:
                os.remove(file_path)
                print(f"Deleted: {file_path}")
            except Exception as e:
                print(f"Error deleting {file_path}: {e}")


if __name__ == "__main__":
    app.run(debug=True)
