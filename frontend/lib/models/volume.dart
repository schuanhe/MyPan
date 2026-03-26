class Volume {
  final int id;
  final String name;
  final String remark;
  final String accessMode;
  final String accessURLKey;

  Volume({
    required this.id,
    required this.name,
    required this.remark,
    required this.accessMode,
    required this.accessURLKey,
  });

  factory Volume.fromJson(Map<String, dynamic> json) {
    return Volume(
      // 后端现在统一返回小写 key
      id: (json['id'] ?? json['ID'] ?? 0) as int,
      name: json['name'] ?? json['Name'] ?? '未命名卷',
      remark: json['remark'] ?? json['Remark'] ?? '',
      accessMode: json['accessMode'] ?? 'private',
      accessURLKey: json['accessURLKey'] ?? '',
    );
  }

  String get publicURL => '/s/$accessURLKey';
}
