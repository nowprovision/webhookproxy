package webhookproxy

import "strings"
import "fmt"
import "net"
import "net/http"

func Protect(ipNetworks []*net.IPNet, handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
		valid := false
		clientIp := net.ParseIP(strings.Split(r.RemoteAddr, ":")[0])

		for _, ipNetwork := range ipNetworks {
			if ipNetwork.Contains(clientIp) {
				valid = true
				break
			}
		}

		if !valid {
			w.WriteHeader(403)
			fmt.Fprintf(w, "%s", "IP not in whitelist")
			return
		}

		handler(w, r)
	}

}
