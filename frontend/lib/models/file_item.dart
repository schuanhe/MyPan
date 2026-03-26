class FileItem {
  final String name;
  final String path;
  final bool isDir;
  final int size;
  final int modTime;

  FileItem({
    required this.name,
    required this.path,
    required this.isDir,
    required this.size,
    required this.modTime,
  });

  factory FileItem.fromJson(Map<String, dynamic> json) {
    return FileItem(
      name: json['name'] ?? 'Unknown',
      path: json['path'] ?? '',
      isDir: json['isDir'] ?? false,
      size: json['size'] ?? 0,
      modTime: json['modTime'] ?? 0,
    );
  }

  String get readableSize {
    if (isDir) return '-';
    if (size < 1024) return '$size B';
    if (size < 1024 * 1024) return '${(size / 1024).toStringAsFixed(1)} KB';
    if (size < 1024 * 1024 * 1024) return '${(size / (1024 * 1024)).toStringAsFixed(1)} MB';
    return '${(size / (1024 * 1024 * 1024)).toStringAsFixed(2)} GB';
  }
}
