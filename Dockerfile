FROM golang:1.19 AS builder
RUN apt-get update && apt-get install -y libzmq3-dev
COPY . /whitenote
RUN cd /whitenote && go build .

FROM jupyter/base-notebook:lab-3.4.4

USER root
RUN apt-get update && apt-get install -y libzmq5 && \
    apt-get clean && rm -rf /var/lib/apt/lists/*
COPY --from=builder /whitenote/whitenote /usr/local/bin/whitenote
COPY ./kernel /usr/local/share/jupyter/kernels/whitenote
COPY ./example.ipynb /home/${NB_USER}/example.ipynb
RUN chmod -R g+w /home/${NB_USER}/*.ipynb && chgrp -R users /home/${NB_USER}/*.ipynb

USER $NB_USER
