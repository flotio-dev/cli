package cmd

import (
	"fmt"

	"github.com/flotio-dev/cli/pkg/api/client/auth"
	"github.com/flotio-dev/cli/pkg/client"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate to Flotio",
	Long: `Authenticate with your email and password.
Tokens are stored in ~/.flotio/auth.json and used automatically by other commands.`,
	Example: `  flotio login --email me@example.com --password s3cret
  flotio login -e me@example.com -p s3cret`,
	RunE: func(cmd *cobra.Command, args []string) error {
		email, _ := cmd.Flags().GetString("email")
		password, _ := cmd.Flags().GetString("password")

		if email == "" || password == "" {
			return fmt.Errorf("--email and --password are required")
		}

		params := auth.NewPostAuthLoginParams().WithLogin(
			&LoginRequest{Email: email, Password: password},
		)
		resp, err := api.Auth.PostAuthLogin(params)
		if err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

		payload := resp.GetPayload()
		if err := client.SaveTokens(payload.AccessToken, payload.RefreshToken); err != nil {
			return fmt.Errorf("saving tokens: %w", err)
		}

		fmt.Println("✓ Logged in successfully")
		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use:     "logout",
	Short:   "Log out from Flotio",
	Example: `  flotio logout`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tokens, err := client.LoadTokens()
		if err != nil || tokens == nil {
			fmt.Println("Not logged in")
			return nil
		}

		params := auth.NewPostAuthLogoutParams().WithLogout(
			&RefreshTokenRequest{RefreshToken: tokens.RefreshToken},
		)
		_, _ = api.Auth.PostAuthLogout(params)

		if err := client.ClearTokens(); err != nil {
			return fmt.Errorf("clearing tokens: %w", err)
		}
		fmt.Println("✓ Logged out")
		return nil
	},
}

var whoamiCmd = &cobra.Command{
	Use:     "whoami",
	Short:   "Show the currently authenticated user",
	Example: `  flotio whoami`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in — run 'flotio login' first")
		}

		user, err := client.WhoAmI(cfg.ResolveHost())
		if err != nil {
			return fmt.Errorf("whoami failed: %w", err)
		}

		fmt.Printf("email:    %v\n", user["email"])
		fmt.Printf("username: %v\n", user["username"])
		fmt.Printf("id:       %v\n", user["id"])
		return nil
	},
}

func init() {
	loginCmd.Flags().StringP("email", "e", "", "Email address")
	loginCmd.Flags().StringP("password", "p", "", "Password")

	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(whoamiCmd)
}
