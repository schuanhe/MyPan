import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:flutter/services.dart';
import 'package:intl/intl.dart';
import '../../providers/share_provider.dart';
import '../../services/api_service.dart';

class ShareManagementPage extends StatefulWidget {
  const ShareManagementPage({super.key});

  @override
  State<ShareManagementPage> createState() => _ShareManagementPageState();
}

class _ShareManagementPageState extends State<ShareManagementPage> {
  String _searchQuery = '';
  String _filterType = 'all'; // all, volume, file, directory

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<ShareProvider>().fetchShares();
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('分享管理中心', style: TextStyle(fontWeight: FontWeight.bold)),
        centerTitle: false,
        elevation: 0,
        backgroundColor: Colors.white,
        foregroundColor: Colors.black,
      ),
      body: Consumer<ShareProvider>(
        builder: (context, shareProv, _) {
          if (shareProv.isLoading) {
            return const Center(child: CircularProgressIndicator());
          }

          final filteredShares = shareProv.shares.where((s) {
            final baseUrl = ApiService().dio.options.baseUrl.replaceAll('/api', '');
            final typePath = s.type == 'volume' ? 's' : 'f';
            final shareUrl = '$baseUrl/$typePath/${s.accessURLKey}';
            
            final matchesSearch = s.name.toLowerCase().contains(_searchQuery.toLowerCase()) ||
                                 shareUrl.toLowerCase().contains(_searchQuery.toLowerCase());
            final matchesType = _filterType == 'all' || s.type == _filterType;
            return matchesSearch && matchesType;
          }).toList();

          // 按卷名排序
          filteredShares.sort((a, b) => a.volumeName.compareTo(b.volumeName));

          return Column(
            children: [
              _buildFilterBar(),
              Expanded(
                child: filteredShares.isEmpty
                    ? _buildEmptyState()
                    : ListView.separated(
                        padding: const EdgeInsets.all(24),
                        itemCount: filteredShares.length,
                        separatorBuilder: (context, index) => const SizedBox(height: 12),
                        itemBuilder: (context, index) {
                          return _buildShareCard(context, filteredShares[index]);
                        },
                      ),
              ),
            ],
          );
        },
      ),
    );
  }

  Widget _buildFilterBar() {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 12),
      color: Colors.white,
      child: Row(
        children: [
          Expanded(
            child: TextField(
              decoration: InputDecoration(
                hintText: '搜索分享名称...',
                prefixIcon: const Icon(Icons.search),
                contentPadding: const EdgeInsets.symmetric(vertical: 0),
                border: OutlineInputBorder(borderRadius: BorderRadius.circular(12)),
              ),
              onChanged: (val) => setState(() => _searchQuery = val),
            ),
          ),
          const SizedBox(width: 16),
          DropdownButton<String>(
            value: _filterType,
            underline: const SizedBox(),
            items: const [
              DropdownMenuItem(value: 'all', child: Text('全部类型')),
              DropdownMenuItem(value: 'volume', child: Text('储存卷')),
              DropdownMenuItem(value: 'directory', child: Text('文件夹')),
              DropdownMenuItem(value: 'file', child: Text('文件')),
            ],
            onChanged: (val) => setState(() => _filterType = val!),
          ),
        ],
      ),
    );
  }

  Widget _buildEmptyState() {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(Icons.link_off, size: 64, color: Colors.grey[300]),
          const SizedBox(height: 16),
          Text(_searchQuery.isEmpty ? '暂无分享项' : '未找到匹配的分享', style: const TextStyle(color: Colors.grey)),
        ],
      ),
    );
  }

  Widget _buildShareCard(BuildContext context, ShareItem share) {
    final baseUrl = ApiService().dio.options.baseUrl.replaceAll('/api', '');
    final typePath = share.type == 'volume' ? 's' : 'f';
    final shareUrl = '$baseUrl/$typePath/${share.accessURLKey}';
    final isExpired = share.expiresAt != null && share.expiresAt!.isBefore(DateTime.now());

    return Card(
      child: ExpansionTile(
        leading: Icon(
          share.type == 'volume' ? Icons.storage_rounded : (share.type == 'directory' ? Icons.folder_rounded : Icons.insert_drive_file_rounded),
          color: const Color(0xFF4F46E5),
        ),
        title: Text(share.name, style: const TextStyle(fontWeight: FontWeight.bold)),
        subtitle: Text(
          '${share.type == 'volume' ? "卷" : (share.type == 'directory' ? "目录" : "文件")} | ${share.accessURLKey}',
          style: TextStyle(color: Colors.grey[500], fontSize: 12),
        ),
        trailing: _buildStatusChip(share.accessMode, isExpired),
        children: [
          Padding(
            padding: const EdgeInsets.all(16.0),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const Divider(),
                const Text('详细信息', style: TextStyle(fontSize: 12, fontWeight: FontWeight.bold, color: Colors.grey)),
                const SizedBox(height: 8),
                Row(
                  children: [
                    const Icon(Icons.link, size: 16, color: Colors.grey),
                    const SizedBox(width: 8),
                    Expanded(child: SelectableText(shareUrl, style: const TextStyle(color: Colors.blue))),
                    IconButton(
                      icon: const Icon(Icons.copy_rounded, size: 16),
                      onPressed: () {
                        Clipboard.setData(ClipboardData(text: shareUrl));
                        ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('链接已复制')));
                      },
                    ),
                  ],
                ),
                if (share.path != "/") ...[
                  Row(
                    children: [
                      const Icon(Icons.folder_open, size: 16, color: Colors.grey),
                      const SizedBox(width: 8),
                      Text('原始路径: ${share.path}', style: const TextStyle(fontSize: 12)),
                    ],
                  ),
                ],
                Row(
                  children: [
                    const Icon(Icons.timer_outlined, size: 16, color: Colors.grey),
                    const SizedBox(width: 8),
                    Text(
                      '有效期: ${share.expiresAt == null ? "永久有效" : DateFormat('yyyy-MM-dd HH:mm').format(share.expiresAt!)}',
                      style: TextStyle(fontSize: 12, color: isExpired ? Colors.red : Colors.black87),
                    ),
                  ],
                ),
                const SizedBox(height: 16),
                Row(
                  mainAxisAlignment: MainAxisAlignment.end,
                  children: [
                    if (share.type != 'volume') // 仅文件/文件夹支持在此处快速编辑
                      TextButton.icon(
                        onPressed: () => _showEditDialog(context, share),
                        icon: const Icon(Icons.edit_note_rounded),
                        label: const Text('修改配置'),
                      ),
                    const SizedBox(width: 8),
                    ElevatedButton.icon(
                      onPressed: () async {
                        final confirm = await _showRevokeConfirm(context);
                        if (confirm == true) {
                          await context.read<ShareProvider>().revokeShare(share.type, share.id);
                        }
                      },
                      icon: const Icon(Icons.link_off_rounded, size: 18),
                      label: const Text('撤销分享'),
                      style: ElevatedButton.styleFrom(backgroundColor: Colors.red[50], foregroundColor: Colors.red, elevation: 0),
                    ),
                  ],
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildStatusChip(String mode, bool isExpired) {
    if (isExpired) {
      return Container(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
        decoration: BoxDecoration(color: Colors.red[50], borderRadius: BorderRadius.circular(6)),
        child: const Text('已过期', style: TextStyle(color: Colors.red, fontSize: 10, fontWeight: FontWeight.bold)),
      );
    }
    Color color = Colors.green;
    String text = '开放';
    if (mode == 'password') {
      color = Colors.orange;
      text = '保护';
    } else if (mode == 'login') {
      color = Colors.blue;
      text = '登录';
    }
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(color: color.withOpacity(0.1), borderRadius: BorderRadius.circular(6)),
      child: Text(text, style: TextStyle(color: color, fontSize: 10, fontWeight: FontWeight.bold)),
    );
  }

  void _showEditDialog(BuildContext context, ShareItem share) {
    final pwdCtrl = TextEditingController();
    final keyCtrl = TextEditingController(text: share.accessURLKey);
    int days = 0; // 0=no change, -1=set permanent, >0=new expire from now
    String accessMode = share.accessMode; // initialize with current mode

    showDialog(
      context: context,
      builder: (ctx) => StatefulBuilder(
        builder: (context, setStateSB) => AlertDialog(
          title: const Text('修改分享配置'),
          content: SingleChildScrollView(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                DropdownButtonFormField<String>(
                  value: accessMode,
                  decoration: const InputDecoration(labelText: '访问权限', prefixIcon: Icon(Icons.security_rounded)),
                  items: const [
                    DropdownMenuItem(value: 'public', child: Text('公开访问')),
                    DropdownMenuItem(value: 'password', child: Text('密码保护')),
                    DropdownMenuItem(value: 'login', child: Text('仅登录可见')),
                  ],
                  onChanged: (val) => setStateSB(() => accessMode = val!),
                ),
                if (accessMode == 'password') ...[
                  const SizedBox(height: 16),
                  TextField(
                    controller: pwdCtrl,
                    decoration: const InputDecoration(labelText: '新提取码 (留空不修改)', prefixIcon: Icon(Icons.lock_outline)),
                  ),
                ],
                const SizedBox(height: 16),
                TextField(
                  controller: keyCtrl,
                  decoration: const InputDecoration(labelText: '访问短码', prefixIcon: Icon(Icons.link_rounded)),
                ),
                const SizedBox(height: 16),
                DropdownButtonFormField<int>(
                  value: days,
                  decoration: const InputDecoration(labelText: '重设有效期'),
                  items: const [
                    DropdownMenuItem(value: 0, child: Text('保持不变')),
                    DropdownMenuItem(value: -1, child: Text('改为永久有效')),
                    DropdownMenuItem(value: 1, child: Text('延长 1 天')),
                    DropdownMenuItem(value: 7, child: Text('延长 7 天')),
                    DropdownMenuItem(value: 30, child: Text('延长 30 天')),
                  ],
                  onChanged: (val) => setStateSB(() => days = val!),
                ),
              ],
            ),
          ),
          actions: [
            TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('取消')),
            ElevatedButton(
              onPressed: () async {
                final succ = await context.read<ShareProvider>().updateShare(
                  share.id,
                  password: (accessMode == 'password' && pwdCtrl.text.isNotEmpty) ? pwdCtrl.text : null,
                  days: days,
                  accessURLKey: keyCtrl.text == share.accessURLKey ? null : keyCtrl.text,
                  accessMode: accessMode,
                );
                if (mounted && succ) Navigator.pop(ctx);
              },
              child: const Text('确认保存'),
            ),
          ],
        ),
      ),
    );
  }

  Future<bool?> _showRevokeConfirm(BuildContext context) {
    return showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('确认撤销分享？'),
        content: const Text('撤销后，所有人将无法通过原有链接访问。'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('保持')),
          ElevatedButton(
            style: ElevatedButton.styleFrom(backgroundColor: Colors.red, foregroundColor: Colors.white),
            onPressed: () => Navigator.pop(ctx, true), 
            child: const Text('确认'),
          ),
        ],
      ),
    );
  }
}
