package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/robfig/cron/v3"
)

type CronJob struct {
	Schedule string
	Command  string
}

type Options struct {
	quiet    bool
	cronVar  string
	cronPath string
}

var options Options = Options{
	quiet:    false,
	cronVar:  "CRONTAB",
	cronPath: "/etc/crontab",
}

func info(message ...any) {
	if !options.quiet {
		fmt.Println(message...)
	}
}

func parseCronTab(scanner *bufio.Scanner) ([]CronJob, error) {
	var cronTasks []CronJob
	for scanner.Scan() {
		line := scanner.Text()

		// Ignore empty lines and lines starting with a # (comments).
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split the cron line into fields.
		fields := strings.Fields(line)
		if len(fields) < 6 {
			return nil, fmt.Errorf("Invalid cron line: %s", line)
		}

		// Extract the schedule and the command from the fields.
		schedule := strings.Join(fields[0:5], " ")
		command := strings.Join(fields[5:], " ")

		// Add the line to the slice of cron lines.
		cronTasks = append(cronTasks, CronJob{
			Schedule: schedule,
			Command:  command,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("Error reading cron tab: %v", err)
	}
	return cronTasks, nil
}

func readCronFile(path string) ([]CronJob, error) {
	// Open the cron file.
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []CronJob{}, nil
		}
		return nil, fmt.Errorf("Error opening cron file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	return parseCronTab(scanner)
}

func readCronVar(variable string) ([]CronJob, error) {
	value, found := os.LookupEnv(variable)
	if found {
		scanner := bufio.NewScanner(strings.NewReader(value))
		return parseCronTab(scanner)
	}
	return []CronJob{}, nil
}

func addJobsFromFile(path string, cronJobs *[]CronJob) error {
	jobs, err := readCronFile(path)
	if err != nil {
		return err
	}
	*cronJobs = append(*cronJobs, jobs...)
	return nil
}

func addJobsFromPath(path string, cronJobs *[]CronJob) error {
	stat, err := os.Stat(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	if stat.IsDir() {
		// Read the files in the directory
		files, err := os.ReadDir(path)
		if err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("Error reading %s directory: %v", path, err)
			}
		}
		// Iterate files in directory
		for _, file := range files {
			if file.IsDir() {
				continue
			}

			// Add the jobs from this file in the directory
			err := addJobsFromFile(filepath.Join(path, file.Name()), cronJobs)
			if err != nil {
				return err
			}
		}
	} else {
		// Add the jobs from the file at `path`
		err := addJobsFromFile(path, cronJobs)
		if err != nil {
			return err
		}
	}
	return nil
}

// Adds any cron jobs found at the environment variable to the slice
func addJobsFromVar(variable string, cronJobs *[]CronJob) error {
	jobs, err := readCronVar(variable)
	if err != nil {
		return err
	}
	*cronJobs = append(*cronJobs, jobs...)
	return nil
}

func getCronJobs() ([]CronJob, error) {
	var jobs []CronJob

	// Add the jobs from the `cronPath` option
	addJobsFromPath(options.cronPath, &jobs)

	// Add the jobs from the `cronVar` option
	addJobsFromVar(options.cronVar, &jobs)

	return jobs, nil
}

func execCronJob(job CronJob) error {
	// Set up the command to be run.
	cmd := exec.Command("/bin/sh", "-c", job.Command)

	// Set the environment variables of the command to be the same as the current process.
	cmd.Env = os.Environ()

	// Redirect the command's stdout and stderr to our stdout and stderr.
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command.
	return cmd.Run()
}

func scheduleCronJob(job CronJob, c *cron.Cron) (cron.EntryID, error) {
	// Add the job to the cron scheduler.
	return c.AddFunc(job.Schedule, func() {
		// Run the command.
		err := execCronJob(job)
		if err != nil {
			fmt.Printf("Error running command: %v\n", err)
		}
	})
}

func scheduleCronJobs(c *cron.Cron) {
	// Fetch all valid cron lines
	jobs, err := getCronJobs()
	if err != nil {
		fmt.Printf("Could not get cron jobs: %v\n", err)
		os.Exit(1)
	}

	// Schedule all cron jobs
	for _, job := range jobs {
		info("Scheduling cron job:", job.Schedule, job.Command)

		_, err := scheduleCronJob(job, c)
		if err != nil {
			fmt.Printf("Warning: Could not schedule cron job: %v\n", err)
		}
	}
}

func main() {
	// Initialize
	c := cron.New()

	// Schedule all jobs
	scheduleCronJobs(c)

	// Start the cron scheduler.
	c.Start()

	// Set up a signal handler to handle the SIGINT and SIGTERM signals.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for a signal
	<-sigChan

	// Shut down the cron scheduler
	fmt.Printf("Shutting down...")
	<-c.Stop().Done()
}
