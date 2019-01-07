# Tableprov CSV Delimiting Serializer Plugin

The `tableprov_csv_delimiting` serializer uses protobuf to wrap a table snapshot chunk. Additionally, it adds a 4-byte Integer denoting how large the entire packet is:

packet_length
TableSnapshotChunk{
	Version
	ChunkIdentifier{
		SnapshotIdentifier {
			TableIdentifier{
				NetworkName
				TableName
			}
			PublisherIdentifier{
				Type
				Identifier
			}
			PublicationTimestamp{
				Time
			}
		}
		ChunkSequenceNumber
		IsLastInSnapshot
	}
	SnapshotProperties{
		IsPresent
		EncodingMetadata{
			Encoding
		}
		WindowingSemantics
	}
	Data
}
