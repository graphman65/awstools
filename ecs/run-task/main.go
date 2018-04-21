package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/hamstah/awstools/common"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	flags          = common.KingpinSessionFlags()
	taskDefinition = kingpin.Flag("task-definition", "ECS task definition").Required().String()
	cluster        = kingpin.Flag("cluster", "ECS cluster").Required().String()
)

func main() {
	kingpin.CommandLine.Name = "ecs-run-task"
	kingpin.CommandLine.Help = "Run a task on ECS."
	kingpin.Parse()

	session := session.Must(session.NewSession())
	conf := common.AssumeRoleConfig(flags, session)

	ecsClient := ecs.New(session, conf)

	_, err := ecsClient.RunTask(&ecs.RunTaskInput{
		TaskDefinition: taskDefinition,
		Cluster:        cluster,
		Count:          aws.Int64(1),
	})
	common.FatalOnError(err)
}
