package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type DMARCTag struct {
	Tag   string
	Value string
}

type DMARCRecord struct {
	Tags map[string]string
}

func main() {
	debug := flag.Bool("debug", false, "enable debug logging")
	flag.Parse()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	}
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		<-signalChan
		os.Exit(0)
	}()

	reader := bufio.NewScanner(os.Stdin)

	for reader.Scan() {
		domain := reader.Text()
		domain = strings.TrimSpace(domain)
		if domain == "" {
			continue
		}

		log.Info().Str("domain", domain).Msg("Processing domain")

		dmarcRecord, err := net.LookupTXT("_dmarc." + domain)
		if err != nil {
			log.Warn().Err(err).Str("domain", domain).Msg("Failed to lookup DMARC TXT record")
			continue
		}

		log.Debug().Strs("records", dmarcRecord).Msg("DMARC TXT records found")

		for _, record := range dmarcRecord {
			record = strings.TrimSpace(record)
			record = strings.ToLower(record)

			log.Debug().Str("record", record).Msg("Parsing DMARC record")
			dmarcParsed, err := parseDMARCRecord(record)
			if err != nil {
				log.Warn().Err(err).Str("record", record).Msg("Failed to parse DMARC record")
				continue
			}

			log.Debug().Interface("parsed_record", dmarcParsed).Msg("Parsed DMARC record")

			ruaValue, ok := dmarcParsed.Tags["rua"]
			if !ok {
				log.Warn().Str("domain", domain).Msg("No 'rua' tag found in DMARC record")
				continue
			}

			log.Debug().Str("rua", ruaValue).Msg("Found 'rua' tag")

			afterAt := strings.Split(ruaValue, "@")
			if len(afterAt) != 2 {
				log.Warn().Str("rua", ruaValue).Msg("Invalid 'rua' email address")
				continue
			}

			emailDomain := afterAt[1]
			fmt.Println(emailDomain)
			log.Info().Str("email_domain", emailDomain).Msg("Extracted email domain from 'rua' tag")
		}
	}
}

func parseDMARCRecord(record string) (DMARCRecord, error) {
	dmarcRecord := DMARCRecord{
		Tags: make(map[string]string),
	}

	tags := strings.Split(record, ";")
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		dmarcTag, err := parseDMARCTag(tag)
		if err != nil {
			log.Warn().Err(err).Str("tag", tag).Msg("Failed to parse DMARC tag")
			return DMARCRecord{}, err
		}

		log.Debug().Str("tag", dmarcTag.Tag).Str("value", dmarcTag.Value).Msg("Parsed DMARC tag")

		dmarcRecord.Tags[dmarcTag.Tag] = dmarcTag.Value
	}

	return dmarcRecord, nil
}

func parseDMARCTag(tag string) (DMARCTag, error) {
	tagSplit := strings.SplitN(tag, "=", 2)
	if len(tagSplit) != 2 {
		err := fmt.Errorf("invalid tag: %s", tag)
		log.Warn().Err(err).Str("tag", tag).Msg("Failed to parse DMARC tag")
		return DMARCTag{}, err
	}

	return DMARCTag{
		Tag:   strings.TrimSpace(tagSplit[0]),
		Value: strings.TrimSpace(tagSplit[1]),
	}, nil
}
