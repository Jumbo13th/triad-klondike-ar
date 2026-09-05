// Command klondiked is the local backend of Triad: Klondike. It is the system of
// record for every logical record the game mode keeps (players, wallets, ledger,
// receipts, audit, runtime configuration) and answers the game server over the
// loopback interface only.
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Jumbo13th/triad-klondike-ar/backend/internal/api"
	"github.com/Jumbo13th/triad-klondike-ar/backend/internal/domain"
	"github.com/Jumbo13th/triad-klondike-ar/backend/internal/store"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:8471", "address to listen on; must be a loopback address")
	dataDir := flag.String("data", "data", "directory holding the database and the klondiked.json seed")
	announce := flag.String("announce-contract", domain.ContractVersion, "contract version announced by /v1/health (demo-world control)")
	delay := flag.Duration("delay", 0, "delay every command answer by this duration; health and reads stay instant (demo-world control)")
	flag.Parse()

	if err := requireLoopback(*listen); err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(*dataDir, 0o755); err != nil {
		log.Fatal(err)
	}

	st, err := store.Open(filepath.Join(*dataDir, "klondike.db"), filepath.Join(*dataDir, "klondiked.json"))
	if err != nil {
		log.Fatal(err)
	}
	defer st.Close()

	handler := api.New(domain.NewService(st), *announce, *delay)
	server := &http.Server{
		Addr:              *listen,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("klondiked contract %s listening on %s (announcing %s, delay %s, data %s)",
		domain.ContractVersion, *listen, *announce, *delay, *dataDir)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

// requireLoopback refuses any address that could be reached from another host.
func requireLoopback(addr string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("listen address %q: %w", addr, err)
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("listen address %q is not a loopback address; the backend never listens on a public interface", addr)
	}
	return nil
}
