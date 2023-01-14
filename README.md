# gh-comment-update
Update existing self (assume bot account) comment that find by regexp.

## Why I create this

`gh issue comment` command only have `--edit-last` option to update existing comment.

Motivate to update specific comment is reason why I create this.

## Usage

```bash
gh comment-update --issue <number> --regexp 'regexp to determine which comments to update' --body 'body of comment'
```
