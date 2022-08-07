package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type VolumeInfo struct {
    Name string
    ContainerName string
    VolumePath string
    S3BackupBucket string
}

func runDockerCommand(args []string) {
    path, err := exec.LookPath("docker")

	if err != nil {
		log.Fatalf("Cannot find path %s", err)
	}

    cmd := exec.Cmd {
        Path: path,
        Args: append([]string{ path }, args...),
        Stdin: os.Stdin,
        Stdout: os.Stdout,
    }
    fmt.Println("The command is", cmd)

    err = cmd.Run()
	if err != nil {
		log.Fatalf("Cannot find path %s", err)
	}
}

func main() {
    if len(os.Args) <= 1 {
        log.Fatalf("You have to pass the container name")
    }

    volumeName := os.Args[1]

	err := godotenv.Load(".env")
	
	if err != nil {
		log.Fatalf("Cannot load env file %s", err)
	}

    volumesFile, err := os.ReadFile("volumes.json")

	if err != nil {
		log.Fatalf("Cannot read volumes.json file %s", err)
	}

    var volumes []VolumeInfo

    err = json.Unmarshal(volumesFile, &volumes)

	if err != nil {
		log.Fatalf("Cannot decrypt volumes from file %s", err)
	}

    volumesByName := make(map[string]VolumeInfo)
    fmt.Println(volumes)

    for _, volumeInfo := range volumes {
        fmt.Println("Current container name", volumeInfo.Name)
        volumesByName[volumeInfo.Name] = volumeInfo
    }

    currentVolume, present := volumesByName[volumeName]
    fmt.Println(volumesByName)

    if !present {
        log.Fatalf("The container name does not exist in the volumes.json file: %s", volumeName)
    }

    bucketName := os.Getenv("AWS_S3_BUCKET_NAME")

	fmt.Println("Success loading the env file")

	minioClient, err := minio.New("s3.amazonaws.com", &minio.Options{
		Creds:  credentials.NewStaticV4(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), ""),
		Secure: true,
	})

	if err != nil {
		log.Fatalf("Cannot instantiate the minio client %s", err)
	}

    fmt.Println("Gonna list object")
	context := context.Background()

	objects := minioClient.ListObjects(context, bucketName, minio.ListObjectsOptions{
		Prefix:    currentVolume.S3BackupBucket,
		Recursive: true,
	})

	objectNames := []string{}

	for object := range objects {
		if object.Err != nil {
			fmt.Println("Error with object", object.Err)
			return
		}
		objectNames = append(objectNames, object.Key)
	}

	sort.Sort(sort.Reverse(sort.StringSlice(objectNames)))

	latestObject := objectNames[0]
    splittedNames := strings.Split(latestObject, "/")
    fileName := splittedNames[len(splittedNames) - 1] 

    fmt.Println("The object name to retrieve is", fileName)

    object, err := minioClient.GetObject(context, bucketName, latestObject, minio.GetObjectOptions{})

	if err != nil {
		log.Fatalf("Cannot get object %s", err)
	}

    fmt.Println("Creating the '/tmp/docker-backup' directory")
    os.MkdirAll("/tmp/docker-backup", os.ModePerm)
    localFile, err := os.Create(fmt.Sprintf("/tmp/docker-backup/%s", fileName))

	if err != nil {
		log.Fatalf("Cannot create file %s", err)
	}

    if _, err = io.Copy(localFile, object); err != nil {
		log.Fatalf("Cannot copy the file %s", err)
    }

    fmt.Println("Successfuly download the object")

    fmt.Println("Creating the container")
    runDockerCommand([]string{"run", "--rm", "-d", "--volumes-from", currentVolume.ContainerName, "-v", "/tmp/docker-backup:/home", "--name", "restoration-volume-container", "ubuntu", "sleep", "infinity"})

    fmt.Println("Going in the container")
    runDockerCommand([]string{"exec", "-it", "restoration-volume-container", "tar", "-zxvf", fmt.Sprintf("/home/%s", fileName), "-C", currentVolume.VolumePath, "--strip-components", "1"})

    fmt.Println("Stoping the container")
    runDockerCommand([]string{"stop", "restoration-volume-container"})

    fmt.Println("The volume has been restored")
}

