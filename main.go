package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/cloudflare/cloudflare-go"
	"github.com/miekg/dns"
	"github.com/russellchadwick/configurationservice"
	"net"
)

func main() {

	client := configurationservice.Client{}
	cloudflareApiKey, err := client.GetConfiguration("cloudflare/apikey")
	if err != nil {
		log.WithField("error", err).Panic("unable to get configuration")
	}
	cloudflareEmail, err := client.GetConfiguration("cloudflare/email")
	if err != nil {
		log.WithField("error", err).Panic("unable to get configuration")
	}

	dnsIp := dnsIpAddress("russellchadwick.com")
	log.WithField("ip", dnsIp).Info("current dns address")

	myIp := myIpAddress()
	log.WithField("ip", myIp).Info("my ip address")

	if !dnsIp.Equal(myIp) {
		log.Info("ip address is different, time to update dns")
		updateCloudflare(cloudflareApiKey, cloudflareEmail, myIp)
	}

	log.Info("done")

}

func updateCloudflare(cloudflareApiKey, cloudflareEmail *string, ip net.IP) {
	api := cloudflare.New(*cloudflareApiKey, *cloudflareEmail)

	zones, err := api.ListZones()
	if err != nil {
		log.WithField("error", err).Panic("unable to list zones")
	}

	for _, zone := range zones {
		log.WithField("zone", zone).Info("found a zone")

		rr := cloudflare.DNSRecord{}
		records, err := api.DNSRecords(zone.Name, rr)
		if err != nil {
			log.WithField("error", err).WithField("zone", zone).Panic("unable to list zones")
		}

		for _, record := range records {
			log.WithField("record", record).Info("found a dns record")
			if record.Type == "A" && record.Name == zone.Name {
				log.WithField("ip", ip).Info("updating dns")
				record.Content = ip.String()
				api.UpdateDNSRecord(zone.Name, record.ID, record)
			}
		}
	}
}

func myIpAddress() net.IP {
	target := "myip.opendns.com"
	server := "resolver1.opendns.com"

	client := dns.Client{}
	requestMessage := dns.Msg{}
	requestMessage.SetQuestion(target+".", dns.TypeA)
	responseMessage, _, err := client.Exchange(&requestMessage, server+":53")
	if err != nil {
		log.WithField("error", err).Panic("unable to dial dns server")
	}

	if len(responseMessage.Answer) == 0 {
		log.WithField("response", responseMessage).Panic("no results from dns server")
	}

	for _, answer := range responseMessage.Answer {
		switch answer.(type) {
		case *dns.A:
			answerA := answer.(*dns.A)
			return answerA.A
		}
	}

	log.Panic("unable to enumerate answer from dns server")
	return nil
}

func dnsIpAddress(host string) net.IP {
	ips, err := net.LookupIP(host)
	if err != nil {
		log.WithField("error", err).Panic("error from net lookup")
	}

	for _, ip := range ips {
		return ip
	}

	log.Panic("unable find a dns ip address")
	return nil
}