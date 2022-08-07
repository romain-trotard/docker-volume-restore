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

type ContainerInfo struct {
    Name string
    VolumePath string
    S3BackupBucket string
}

type Containers struct {
    containers []ContainerInfo
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

    containerName := os.Args[1]

	err := godotenv.Load(".env")
	
	if err != nil {
		log.Fatalf("Cannot load env file %s", err)
	}

    containersFile, err := os.ReadFile("containers.json")

	if err != nil {
		log.Fatalf("Cannot read containers.json file %s", err)
	}

    var containers []ContainerInfo

    err = json.Unmarshal(containersFile, &containers)

	if err != nil {
		log.Fatalf("Cannot decrypt containers from file %s", err)
	}

    containersByName := make(map[string]ContainerInfo)
    fmt.Println(containers)

    for _, containerInfo := range containers {
        fmt.Println("Current container name", containerInfo.Name)
        containersByName[containerInfo.Name] = containerInfo
    }

    currentContainer, present := containersByName[containerName]
    fmt.Println(containersByName)

    if !present {
        log.Fatalf("The container name does not exist in the containers.json file: %s", containerName)
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
		Prefix:    currentContainer.S3BackupBucket,
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
    runDockerCommand([]string{"run", "--rm", "-d", "--volumes-from", currentContainer.Name, "-v", "/tmp/docker-backup:/home", "--name", "restoration-volume-container", "ubuntu", "sleep", "infinity"})

    fmt.Println("Going in the container")
    runDockerCommand([]string{"exec", "-it", "restoration-volume-container", "tar", "-zxvf", fmt.Sprintf("/home/%s", fileName), "-C", currentContainer.VolumePath, "--strip-components", "1"})

    fmt.Println("Stoping the container")
    runDockerCommand([]string{"stop", "restoration-volume-container"})

    fmt.Println("The volume has been restored")
}

