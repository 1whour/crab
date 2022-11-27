# git filter-branch --env-filter '
# WRONG_EMAIL="guonaihong@qq.com"
# NEW_NAME="gnh123"
# NEW_EMAIL="710390515@qq.com"
# 
# if [ "$GIT_COMMITTER_EMAIL" = "$WRONG_EMAIL" ]
# then
#     export GIT_COMMITTER_NAME="$NEW_NAME"
#     export GIT_COMMITTER_EMAIL="$NEW_EMAIL"
# fi
# 
# if [ "$GIT_AUTHOR_EMAIL" = "$WRONG_EMAIL" ]
# then
#     export GIT_AUTHOR_NAME="$NEW_NAME"
#     export GIT_AUTHOR_EMAIL="$NEW_EMAIL"
# fi
# ' --tag-name-filter cat -- --branches --tags

#!/bin/sh
 
git filter-branch --env-filter '
 
OLD_EMAIL="guonaihong@qq.com"
CORRECT_NAME="gnh123"
CORRECT_EMAIL="710390515@qq.com"

if [ "$GIT_COMMITTER_EMAIL" = "$OLD_EMAIL" ]
then
    export GIT_COMMITTER_NAME="$CORRECT_NAME"
    export GIT_COMMITTER_EMAIL="$CORRECT_EMAIL"
fi
if [ "$GIT_AUTHOR_EMAIL" = "$OLD_EMAIL" ]
then
    export GIT_AUTHOR_NAME="$CORRECT_NAME"
    export GIT_AUTHOR_EMAIL="$CORRECT_EMAIL"
fi
' --tag-name-filter cat -- --branches --tags

