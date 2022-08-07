# docker-volume-restore

## What is it?

This repository is meant to restore docker volume that are stored on AWS S3. It's easy and fast, 
you just have to create `.env` and `volumes.json` files corresponding to your configuration.

> **Note:** Handles named volume and host volume.


## Prerequisite

For now, you have to:
- clone the repository on your pc
- have `go` installed, my version is currently 1.19
- backups must have a date in the file name (useful for sort)

Then you have to create the following files.

### .env

You can copy the `.env-example` presents in the repository and fill the values:
- `AWS_S3_BUCKET_NAME`: the root bucket name of your s3
- `AWS_ACCESS_KEY_ID`: an Access Key ID with read access to the bucket
- `AWS_SECRET_ACCESS_KEY`: a Access Key corresponding to the previous Access Key ID

### volumes.json

This file corresponds to your volumes configurations. It's here that you need to put all the docker 
container configuration that you want to be able to restore.

You can copy the `containers-example.json` file. 

A volume is described by the following property:
- `Name`: an identifier for the volumes. Useful to know which volume to restore
- `ContainerName`: the name of the container that uses the volume
- `VolumePath`: the path where the directory is mounted in the container
- `S3BackupBucket`: the sub-bucket name where are stored backups

> **Note:** What is the difference between `AWS_S3_BUCKET_NAME` and `S3BackupBucket`? 
> `AWS_S3_BUCKET_NAME` corresponds to the root bucket name, for example `rootBucketName`. And 
> `S3BackupBucket` is the sub-bucket name that are stored backups to that volumes, for example `volume1BucketName`.
> 
> Basicaly, for the example volume1 backups are store in `rootBucketName/volume1BucketName` bucket.
> The idea is that you store different volumes backup in different buckets.

## How to launch the script?

For now, you can only run one restoration at a time. To do that you have to determine the volume you want to restore.

For example, if you want to restore the volume named `volume1` just launch the command:

```bash
go run . volume1
```

