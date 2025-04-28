<?php

require_once "side.php";

echo "Hello, php!";

?>
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <title>Polygonal Cat with CTV Effect</title>
  <style>
    /* Fill the screen with a black background and center the canvas */
    html, body {
      margin: 0;
      padding: 0;
      background: #000;
      height: 100%;
      display: flex;
      align-items: center;
      justify-content: center;
    }

    canvas {
      /* Helps keep edges sharp for retro look */
      image-rendering: pixelated;
    }
  </style>
</head>
<body>
  <canvas id="catCanvas" width="150" height="150"></canvas>

  <script>
    /**
     * Draws the polygonal cat silhouette (similar to your screenshot)
     * with two square eyes in the center.
     * @param {CanvasRenderingContext2D} ctx
     */
    function drawCat(ctx) {
      const w = ctx.canvas.width;
      const h = ctx.canvas.height;

      // 1) Fill background black
      ctx.fillStyle = "black";
      ctx.fillRect(0, 0, w, h);

      // 2) Draw the cat silhouette in white
      ctx.fillStyle = "white";
      ctx.beginPath();
      // Left ear
      ctx.moveTo(64, 64);
      ctx.lineTo(48, 32);   // left ear tip
      ctx.lineTo(80, 48);   // edge back near top

      // Top center
      ctx.lineTo(176, 48);

      // Right ear
      ctx.lineTo(208, 32);  // right ear tip
      ctx.lineTo(192, 64);  // edge back near top

      // Main face down to the bottom
      ctx.lineTo(192, 192);
      ctx.lineTo(64, 192);

      ctx.closePath();
      ctx.fill();

      // 3) Add two black squares for the eyes
      ctx.fillStyle = "black";
      // left eye
      ctx.fillRect(96, 96, 8, 8);
      // right eye
      ctx.fillRect(144, 96, 8, 8);
    }

    /**
     * Applies a simple "CTV" chromatic-aberration–style effect by shifting
     * the red channel slightly right and the blue channel slightly left.
     * @param {CanvasRenderingContext2D} ctx
     */
    function applyCTVEffect(ctx) {
      const width = ctx.canvas.width;
      const height = ctx.canvas.height;
      const imageData = ctx.getImageData(0, 0, width, height);
      const data = imageData.data;

      // Create an output array for the processed pixel data
      const output = new Uint8ClampedArray(data.length);

      // Offsets for red and blue channels (pixels)
      const offsetR = 2;   // shift red to the right by 2
      const offsetB = -2;  // shift blue to the left by 2

      for (let y = 0; y < height; y++) {
        for (let x = 0; x < width; x++) {
          const i = (y * width + x) * 4;

          // --- Compute the "source" pixel for the red channel
          let srcXR = x - offsetR;
          let red = 0;
          if (srcXR >= 0 && srcXR < width) {
            red = data[(y * width + srcXR) * 4 + 0];
          } else {
            red = data[i + 0]; // fallback if offset is out of bounds
          }

          // --- Green channel: no offset
          const green = data[i + 1];

          // --- Compute the "source" pixel for the blue channel
          let srcXB = x - offsetB; // note offsetB is negative
          let blue = 0;
          if (srcXB >= 0 && srcXB < width) {
            blue = data[(y * width + srcXB) * 4 + 2];
          } else {
            blue = data[i + 2];
          }

          // Alpha is unchanged
          const alpha = data[i + 3];

          // Write out our adjusted pixel
          output[i + 0] = red;
          output[i + 1] = green;
          output[i + 2] = blue;
          output[i + 3] = alpha;
        }
      }

      // Copy the processed image back to the canvas
      const outImageData = new ImageData(output, width, height);
      ctx.putImageData(outImageData, 0, 0);
    }

    // Main routine: draw the cat first, then apply the CTV effect
    (function main() {
      const canvas = document.getElementById("catCanvas");
      const ctx = canvas.getContext("2d");

      // 1. Draw the polygonal cat
      drawCat(ctx);

      // 2. After a brief delay, apply the post-processing effect
      setTimeout(() => {
        applyCTVEffect(ctx);
      }, 50);
    })();
  </script>
</body>
</html>
