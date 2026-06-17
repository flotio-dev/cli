package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/flotio-dev/cli/internal/config"
	"github.com/flotio-dev/cli/pkg/client"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [project-id]",
	Short: "Initialize a Flotio project in the current directory",
	Long: `Mark the current directory as a Flotio project by creating a .flotio.yaml file.

Without arguments, launches an interactive setup:
  - Authenticates if needed
  - Lists your existing projects
  - Lets you pick one or create a new one

With a project ID: skips interactive mode, uses that ID directly.`,
	Example: `  flotio init
  flotio init 42`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}

		if pc, dir := config.FindProject(cwd); pc != nil {
			return fmt.Errorf("already a Flotio project (ID %d in %s/.flotio.yaml)", pc.ProjectID, dir)
		}

		// Explicit project ID → fast path
		if len(args) > 0 {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid project ID: %s", args[0])
			}
			return saveInit(cwd, id)
		}

		// Non-interactive? Fall back to prompt
		fi, _ := os.Stdin.Stat()
		if fi.Mode()&os.ModeCharDevice == 0 {
			fmt.Print("Enter project ID: ")
			var input string
			fmt.Scanln(&input)
			id, err := strconv.ParseInt(input, 10, 64)
			if err != nil || id <= 0 {
				return fmt.Errorf("invalid project ID: %s", input)
			}
			return saveInit(cwd, id)
		}

		// --- Interactive TTY flow ---
		r := bufio.NewReader(os.Stdin)
		host := cfg.ResolveHost()

		// 1. Auth
		if !client.IsLoggedIn() {
			fmt.Println("Not logged in. Let's authenticate first.")
		fmt.Println()
			email, password, err := promptLogin(r)
			if err != nil {
				return err
			}
			if err := client.DoLogin(host, email, password); err != nil {
				return fmt.Errorf("login failed: %w", err)
			}
			fmt.Println("✓ Logged in")
		fmt.Println()
		} else {
			fmt.Println("✓ Already authenticated")
			fmt.Println()
		}

		// 2. Fetch projects
		fmt.Println("Fetching your projects...")
		projects, err := fetchProjects(host)
		if err != nil {
			return fmt.Errorf("listing projects: %w", err)
		}

		// 3. Menu
		for {
			fmt.Println()
			if len(projects) > 0 {
				fmt.Println("Your projects:")
				for i, p := range projects {
					fmt.Printf("  [%d] %s (ID: %v)\n", i+1, p["name"], p["id"])
				}
				fmt.Println()
			} else {
				fmt.Println("No projects yet.")
			}
			fmt.Println("  [n] Create a new project")
			fmt.Println("  [q] Quit")
			fmt.Println()

			choice := prompt(r, "Choose an option")
			switch strings.ToLower(choice) {
			case "q", "quit":
				return fmt.Errorf("cancelled")
			case "n", "new":
				name, repo, err := promptNewProject(r)
				if err != nil {
					return err
				}
				id, err := createProject(host, name, repo)
				if err != nil {
					return fmt.Errorf("creating project: %w", err)
				}
				return saveInit(cwd, id)
			default:
				idx, err := strconv.Atoi(choice)
				if err != nil || idx < 1 || idx > len(projects) {
					fmt.Println("Invalid choice.")
					continue
				}
				p := projects[idx-1]
				id, _ := p["id"].(float64)
				return saveInit(cwd, int64(id))
			}
		}
	},
}

func promptLogin(r *bufio.Reader) (email, password string, err error) {
	email = prompt(r, "Email")
	password = prompt(r, "Password")
	if email == "" || password == "" {
		return "", "", fmt.Errorf("email and password are required")
	}
	return email, password, nil
}

func promptNewProject(r *bufio.Reader) (name, repo string, err error) {
	fmt.Println("\nCreate a new project")
	name = prompt(r, "Project name")
	if name == "" {
		return "", "", fmt.Errorf("project name is required")
	}
	repo = prompt(r, "Git repo URL (optional)")
	return name, repo, nil
}

func prompt(r *bufio.Reader, label string) string {
	fmt.Printf("%s: ", label)
	input, _ := r.ReadString('\n')
	return strings.TrimSpace(input)
}

func fetchProjects(host string) ([]map[string]interface{}, error) {
	var raw map[string]interface{}
	if err := client.GetJSON(host, "/project", &raw); err != nil {
		return nil, err
	}
	items, _ := client.ExtractList(raw)
	if items == nil {
		// Try "projects" key
		if projRaw, ok := raw["projects"]; ok {
			if arr, ok := projRaw.([]interface{}); ok {
				items = arr
			}
		}
	}
	var out []map[string]interface{}
	for _, item := range items {
		if m, ok := item.(map[string]interface{}); ok {
			out = append(out, m)
		}
	}
	return out, nil
}

func createProject(host, name, repo string) (int64, error) {
	body := map[string]interface{}{"name": name}
	if repo != "" {
		body["config"] = map[string]interface{}{"git_repo": repo}
	}
	var result map[string]interface{}
	if err := client.PostJSON(host, "/project", body, &result); err != nil {
		return 0, err
	}
	pid, ok := result["id"].(float64)
	if !ok {
		return 0, fmt.Errorf("unexpected response from project creation")
	}
	id := int64(pid)
	fmt.Printf("✓ Project created: [%d] %s\n", id, name)
	return id, nil
}

func saveInit(cwd string, projectID int64) error {
	pc := &config.ProjectConfig{ProjectID: projectID}
	if err := config.SaveProject(cwd, pc); err != nil {
		return err
	}
	fmt.Printf("\n✓ Initialized Flotio project [%d] in %s/\n", projectID, cwd)
	fmt.Println("  Created .flotio.yaml")
	return nil
}

func init() {
	rootCmd.AddCommand(initCmd)
}

// parseProjectID resolves the project ID from a flag value or .flotio.yaml.
func parseProjectID(flagValue string) (int64, error) {
	if flagValue != "" {
		return strconv.ParseInt(flagValue, 10, 64)
	}
	return config.ResolveProjectID(0)
}

// projectIDArg extracts the first arg or resolves from .flotio.yaml.
func projectIDArg(args []string) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}
	id, err := config.ResolveProjectID(0)
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(id, 10), nil
}
