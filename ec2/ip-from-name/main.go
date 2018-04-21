package main

import (
	"fmt"
	"sort"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hamstah/awstools/common"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	flags      = common.KingpinSessionFlags()
	name       = kingpin.Flag("name", "Name of the EC2 instance").Required().String()
	maxResults = kingpin.Flag("max-results", "Max number of IPs to return").Default("9").Int()
)

func main() {
	kingpin.CommandLine.Name = "ec2-ip-from-name"
	kingpin.CommandLine.Help = "Returns a list of instances IP with a given name."
	kingpin.Parse()

	session := session.Must(session.NewSession())
	conf := common.AssumeRoleConfig(flags, session)

	ec2Client := ec2.New(session, conf)
	resp, err := ec2Client.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("tag:Name"),
				Values: []*string{
					aws.String(*name),
				},
			},
		},
	})
	common.FatalOnError(err)

	ips := make([]string, 0, len(resp.Reservations))
	for _, reservation := range resp.Reservations {
		if len(reservation.Instances) > 0 {
			instance := reservation.Instances[0]
			if instance != nil {
				ips = append(ips, *instance.PrivateIpAddress)
			}
		}
	}

	sort.Strings(ips)
	for index, ip := range ips {
		if index >= *maxResults {
			break
		}
		fmt.Println(ip)
	}
}
