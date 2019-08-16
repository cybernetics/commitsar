package cmd

import (
	"errors"
	"fmt"

	"github.com/commitsar-app/commitsar/pkg/history"
	"github.com/commitsar-app/commitsar/pkg/text"
	"github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

func runRoot(cmd *cobra.Command, args []string) error {
	debug := false
	if cmd.Flag("verbose").Value.String() == "true" {
		debug = true
	}

	fmt.Print("Starting analysis of commits on branch\n")

	repo, repoErr := history.Repo(".")

	if repoErr != nil {
		return repoErr
	}

	currentBranch, currentBranchErr := repo.Head()

	if debug {
		fmt.Print("\n[DEBUG] debug mode is on \n")
		fmt.Printf("Current branch %v\n", currentBranch.Name().String())
		refIter, _ := repo.References()

		refIterErr := refIter.ForEach(func(ref *plumbing.Reference) error {
			fmt.Printf("[REF] %v\n", ref.Name().String())
			return nil
		})

		if refIterErr != nil {
			return refIterErr
		}

		fmt.Print("\n")
	}

	if currentBranchErr != nil {
		return currentBranchErr
	}

	commits, commitsErr := history.CommitsOnBranch(repo, currentBranch.Hash(), "origin/master")

	if commitsErr != nil {
		return commitsErr
	}

	var filteredCommits []plumbing.Hash

	for _, commitHash := range commits {
		commitObject, commitErr := repo.CommitObject(commitHash)

		if debug {
			fmt.Printf("\n[DEBUG] Commit found: [hash] %v [message] %v \n", commitObject.Hash, text.MessageTitle(commitObject.Message))
		}

		if commitErr != nil {
			return commitErr
		}

		if !text.IsMergeCommit(commitObject.Message) {
			filteredCommits = append(filteredCommits, commitHash)
		}
	}

	fmt.Printf("\n%v commits filtered out\n", len(commits)-len(filteredCommits))
	fmt.Printf("\nFound %v commit to check\n", len(filteredCommits))

	if len(filteredCommits) == 0 {
		return errors.New(aurora.Red("No commits found, please check you are on a branch outside of main").String())
	}

	var faultyCommits []text.FailingCommit

	for _, commitHash := range filteredCommits {
		commitObject, commitErr := repo.CommitObject(commitHash)

		if commitErr != nil {
			return commitErr
		}

		messageTitle := text.MessageTitle(commitObject.Message)

		textErr := text.CheckMessageTitle(messageTitle)

		if textErr != nil {
			faultyCommits = append(faultyCommits, text.FailingCommit{Hash: commitHash.String(), Message: messageTitle})
		}
	}

	if len(faultyCommits) != 0 {
		failingCommitMessage := text.FormatFailingCommits(faultyCommits)

		fmt.Print(failingCommitMessage)

		fmt.Printf("%v of %v commits are not conventional commit compliant\n", aurora.Red(len(faultyCommits)), aurora.Red(len(commits)))

		return errors.New(aurora.Red("Not all commits are conventiontal commits, please check the commits listed above").String())
	}

	fmt.Print(aurora.Sprintf(aurora.Green("All %v commits are conventional commit compliant\n"), len(filteredCommits)))

	return nil
}
