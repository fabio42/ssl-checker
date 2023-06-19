package domains

import (
	"crypto/tls"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net"
	"os"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/rs/zerolog/log"
)

const (
	DefaultReportFile = "./report.md"
)

type Response struct {
	Domain              string
	Environment         string
	NotBefore, NotAfter time.Time
	Issuer              pkix.Name
	Subject             pkix.Name
	SAN                 []string
	SerialNumber        *big.Int
	Error               error
}

func (i Response) KnownError() string {
	if i.Error != nil {
		switch {
		case strings.HasSuffix(i.Error.Error(), "i/o timeout"):
			return "connexion timeout"
		case strings.HasSuffix(i.Error.Error(), "certificate name does not match input"):
			return "certificate SAN don't include domain"
		case strings.HasSuffix(i.Error.Error(), "no such host"):
			return "no DNS entry for this host"
		case strings.HasPrefix(i.Error.Error(), "tls: failed to verify certificate: x509: certificate is valid for"):
			return "invalid certificate"
		default:
			return i.Error.Error()
		}
	}
	return ""
}

// FilterValue implement the list.Model Item interface
func (i Response) FilterValue() string {
	search := fmt.Sprintf("%v %v %v", i.Domain, i.Issuer, i.NotAfter)
	return search
}

// Title is required by list.Model to display Item main data
func (i Response) Title() string { return i.Domain }

// Description is required by list.Model to display Item description
func (i Response) Description() string { return i.Environment }

func TestDomain(domain, env string, timeO int, out chan<- Response) {
	var resp Response
	log.Debug().Msgf("SSL query for %v", domain)

	nDialer := net.Dialer{
		Timeout: time.Duration(timeO) * time.Second,
	}
	d := tls.Dialer{
		NetDialer: &nDialer,
	}

	conn, err := d.Dial("tcp", domain+":443")
	if err != nil {
		log.Debug().Msgf("Error in d.Dial for domain %s", domain)
		resp = Response{
			Domain:      domain,
			Environment: env,
			Error:       err,
		}
	} else {
		tlsConn := conn.(*tls.Conn)

		resp = Response{
			Domain:       domain,
			Environment:  env,
			NotBefore:    tlsConn.ConnectionState().PeerCertificates[0].NotBefore,
			NotAfter:     tlsConn.ConnectionState().PeerCertificates[0].NotAfter,
			Issuer:       tlsConn.ConnectionState().PeerCertificates[0].Issuer,
			SerialNumber: tlsConn.ConnectionState().PeerCertificates[0].SerialNumber,
			Subject:      tlsConn.ConnectionState().PeerCertificates[0].Subject,
			SAN:          tlsConn.ConnectionState().PeerCertificates[0].DNSNames,
			Error:        err,
		}
		log.Debug().Msgf("SSL query completed for %v", domain)
	}
	out <- resp
}

func CreateReport(domains []Response, queries []string, fileName string, stdOut bool) {
	var file strings.Builder

	headers := []string{"Domain", "Expiration", "Issuer"}

	file.WriteString("# TLS check Domain report\n")
	file.WriteString("\n")

	var domainWidth int
	var issuerWidth int
	for _, i := range domains {
		dSize := utf8.RuneCountInString(i.Domain)
		iSize := utf8.RuneCountInString(i.Issuer.String())
		if dSize > domainWidth {
			domainWidth = dSize
		}
		if iSize > issuerWidth {
			issuerWidth = iSize
		}
		if i.Error != nil {
			eSize := utf8.RuneCountInString(i.Error.Error())
			if eSize > issuerWidth {
				issuerWidth = eSize
			}
		}
	}
	domainWidth = -1 * domainWidth
	issuerWidth = -1 * issuerWidth

	data := map[string][]Response{}
	for _, env := range queries {
		for _, d := range domains {
			if d.Environment != env {
				continue
			}
			data[env] = append(data[env], d)
		}
	}

	now := time.Now()
	for env, domains := range data {
		file.WriteString(fmt.Sprintf("## Domains for %v\n", env))
		file.WriteString("\n")
		file.WriteString(fmt.Sprintf("| %*s | %-10s | %-*s |\n", domainWidth, headers[0], headers[1], issuerWidth, headers[2]))
		file.WriteString(fmt.Sprintf("|-%s-|-%-10s-|-%-*s-|\n", strings.Repeat("-", -1*domainWidth), strings.Repeat("-", 10), issuerWidth, strings.Repeat("-", -1*issuerWidth)))
		sort.Slice(domains, func(i, j int) bool {
			// We want to move errors (null date value) to the end of list
			// adding 99 years there - XXX find a more elegant way
			iDate := domains[i].NotAfter
			jDate := domains[j].NotAfter
			if iDate.IsZero() {
				iDate = now.AddDate(99, 0, 0)
			}
			if jDate.IsZero() {
				jDate = now.AddDate(99, 0, 0)
			}
			return iDate.Before(jDate)
		})

		for _, d := range domains {
			if d.Error != nil {
				file.WriteString(fmt.Sprintf("| %*s | %-10s | %-*s |\n", domainWidth, d.Domain, "NA", issuerWidth, d.Error))
			} else {
				file.WriteString(fmt.Sprintf("| %*s | %-10s | %-*s |\n", domainWidth, d.Domain, d.NotAfter.Format("2006-01-02"), issuerWidth, d.Issuer.String()))
			}

		}
		file.WriteString("\n")
	}

	// We don't save the report on disk if stdOut is requested
	if stdOut {
		fmt.Print(file.String())
	} else {
		err := os.WriteFile(fileName, []byte(file.String()), 0644)
		if err != nil {
			panic(err)
		}
	}
}
