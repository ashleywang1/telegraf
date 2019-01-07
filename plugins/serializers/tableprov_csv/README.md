# Tableprov CSV Serializer Plugin

The `tableprov_csv` serializer uses protobuf to wrap a table snapshot chunk in the following format:

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
