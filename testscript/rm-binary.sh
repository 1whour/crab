# https://vocon-it.com/2018/02/15/git-remove-large-binary-files/
git filter-branch --tag-name-filter 'cat' -f --tree-filter '
    find . -type d -name binarydir | while read dir
      do
        find $dir -type f -name "*.bak" | while read file
          do
             git rm -r -f --ignore-unmatch $file
          done
      done
' -- --all
