package mmapforge

// Magic is the 4-byte file signature written at the start of every mmapforge file.
var Magic = [4]byte{'M', 'M', 'F', 'G'}

// MagicString is the string form of Magic for display purposes.
const MagicString = "MMFG"

// Version is the current binary format version.
const Version uint32 = 1

// HeaderSize is the fixed size of the file header in bytes.
const HeaderSize = 64

// StoreReserveVA is the default virtual address reservation for Store files (1 GB).
const StoreReserveVA = 1 << 30
