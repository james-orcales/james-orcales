
# Bounded Decompression

New_Reader decompresses a zlib stream. It returns an overflow error if the
decompressed stream is larger than the caller's explicit output cap.
