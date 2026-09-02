`atkins docker:build` tags the image with the newest git tag and with `latest`. `atkins docker:push` pushes `latest` every time, and the tag only when HEAD is the commit that tag points at.
