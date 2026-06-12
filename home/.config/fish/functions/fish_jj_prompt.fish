function fish_jj_prompt
    command -sq jj; or return 1
    set -l info (
        jj log --no-graph --ignore-working-copy --color=always --revisions @ --template '
                separate(" ",
                    bookmarks,
                    if(conflict, label("conflict", "conflict")),
                    if(divergent, label("divergent", "divergent")),
                    if(parents.len() > 1, label("merge", "merged")),
                    coalesce(
                        if(empty, label("empty", "empty")),
                        label("change", "change")
                    )
            )
        ' 2>/dev/null
    )
    or return 1

    # Check if there are unpushed changes
    set -l unpushed (
        jj log --no-graph --color=never \
            --revisions '::@- & ~::remote_bookmarks()' \
            --limit 1 | wc -l
    )
    if test $unpushed -gt 0
        set info "$info" (set_color yellow)"↑"(set_color normal)
    end

    test -n "$info"; and printf ' (%s)' $info
end

