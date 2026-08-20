// Command gmail-auth réalise, une seule fois, l'autorisation OAuth2
// "application installée" nécessaire pour que le worker accède à la boîte
// Gmail surveillée (un compte de service seul n'y a pas accès, cf.
// internal/gmailapi) : ouvre une page d'autorisation Google, récupère le
// code de redirection via un petit serveur HTTP local, l'échange contre un
// jeton (incluant un refresh_token qui permet ensuite de se renouveler
// indéfiniment sans nouvelle intervention humaine), et affiche le JSON à
// coller dans .env sous GOOGLE_OAUTH_TOKEN_JSON.
//
// N'est jamais exécuté par le worker lui-même — seulement une fois, à la
// configuration initiale (ou si le jeton est révoqué).
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"golang.org/x/oauth2"

	"github.com/projetimmoai/immo/internal/gmailapi"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "gmail-auth:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := gmailapi.NewOAuthConfigFromEnv()
	if err != nil {
		return err
	}

	// Redirection en boucle locale (127.0.0.1) : l'ancien flux "copier-coller
	// le code" (out-of-band) est désactivé par Google pour les nouveaux
	// clients OAuth2, il faut un serveur HTTP local pour recevoir le code.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("ouverture d'un port local pour recevoir la redirection OAuth2: %w", err)
	}
	defer func() { _ = listener.Close() }()
	port := listener.Addr().(*net.TCPAddr).Port
	cfg.RedirectURL = fmt.Sprintf("http://127.0.0.1:%d", port)

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			code := r.URL.Query().Get("code")
			if code == "" {
				errCh <- fmt.Errorf("pas de code dans la redirection reçue (query=%q)", r.URL.RawQuery)
				if _, err := fmt.Fprintln(w, "Erreur : pas de code reçu, voir le terminal."); err != nil {
					log.Printf("gmail-auth: écriture de la réponse HTTP: %v", err)
				}
				return
			}
			codeCh <- code
			if _, err := fmt.Fprintln(w, "Autorisation reçue, tu peux fermer cet onglet et revenir au terminal."); err != nil {
				log.Printf("gmail-auth: écriture de la réponse HTTP: %v", err)
			}
		}),
	}
	go func() {
		if serveErr := srv.Serve(listener); serveErr != nil && serveErr != http.ErrServerClosed {
			errCh <- serveErr
		}
	}()
	defer func() { _ = srv.Close() }()

	// access_type=offline pour obtenir un refresh_token, prompt=consent pour
	// forcer Google à le renvoyer même si ce compte a déjà autorisé
	// l'application par le passé (sinon, seul un access_token est renvoyé).
	authURL := cfg.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	fmt.Println("1. Ouvrir cette URL dans un navigateur, se connecter avec le compte Gmail à surveiller, et autoriser l'accès :")
	fmt.Println()
	fmt.Println(authURL)
	fmt.Println()
	fmt.Println("En attente de l'autorisation...")

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return err
	}

	tok, err := cfg.Exchange(context.Background(), code)
	if err != nil {
		return fmt.Errorf("échange du code contre un jeton: %w", err)
	}
	if tok.RefreshToken == "" {
		return fmt.Errorf("aucun refresh_token reçu — relancer en révoquant d'abord l'accès existant sur https://myaccount.google.com/permissions")
	}

	tokJSON, err := json.Marshal(tok)
	if err != nil {
		return fmt.Errorf("encodage du jeton: %w", err)
	}

	fmt.Println()
	fmt.Println("2. Coller cette valeur dans .env, variable GOOGLE_OAUTH_TOKEN_JSON :")
	fmt.Println()
	fmt.Println(string(tokJSON))
	return nil
}
