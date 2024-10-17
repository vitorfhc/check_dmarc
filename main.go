package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

type DMARCTag struct {
	Tag   string
	Value string
}

type DMARCRecord struct {
	Tags map[string]string
}

func main() {
	debug := flag.Bool("debug", false, "enable debug mode")
	flag.Parse()

	if *debug {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.FatalLevel)
	}

	logrus.Debug("starting")
	reader := bufio.NewScanner(os.Stdin)

	for reader.Scan() {
		domain := reader.Text()
		domain = strings.TrimSpace(domain)
		if domain == "" {
			continue
		}

		dmarcRecord, err := net.LookupTXT("_dmarc." + domain)
		if err != nil {
			logrus.WithError(err).Errorf("error fetching DMARC record for domain: %q", domain)
			continue
		}

		for _, record := range dmarcRecord {
			dmarcRecord, err := parseDMARCRecord(record)
			if err != nil {
				logrus.WithError(err).Errorf("error parsing DMARC record for domain: %q", domain)
				continue
			}

			logrus.WithField("dmarcRecord", dmarcRecord).Debugf("parsed DMARC record for domain: %q", domain)

			ruaValue, ok := dmarcRecord.Tags["rua"]
			if !ok {
				continue
			}

			afterAt := strings.Split(ruaValue, "@")
			if len(afterAt) != 2 {
				logrus.Errorf("invalid rua tag: %q", ruaValue)
				continue
			}

			emailDomain := afterAt[1]
			fmt.Println(emailDomain)
		}
	}
}

func parseDMARCRecord(record string) (DMARCRecord, error) {
	dmarcRecord := DMARCRecord{
		Tags: make(map[string]string),
	}

	tags := strings.Split(record, ";")
	for _, tag := range tags {
		trimmedTag := strings.TrimSpace(tag)
		dmarcTag, err := parseDMARCTag(trimmedTag)
		if err != nil {
			return DMARCRecord{}, err
		}

		dmarcRecord.Tags[dmarcTag.Tag] = dmarcTag.Value
	}

	return dmarcRecord, nil
}

func parseDMARCTag(tag string) (DMARCTag, error) {
	tagSplit := strings.Split(tag, "=")
	if len(tagSplit) != 2 {
		return DMARCTag{}, fmt.Errorf("invalid tag: %s", tag)
	}

	return DMARCTag{
		Tag:   tagSplit[0],
		Value: tagSplit[1],
	}, nil
}
