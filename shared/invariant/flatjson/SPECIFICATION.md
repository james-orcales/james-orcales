
# Always

`Always` enforces flat JSON encoder assertions without coverage recording. This dependency direction
lets `shared/invariant/default` import the encoder. It does not cause an import cycle or another
default package.
